package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/hyperterse/hyperterse/core/logger"
)

// ScriptRuntime executes bundled route scripts in-process.
type ScriptRuntime struct {
	httpClient *http.Client
}

func NewScriptRuntime() *ScriptRuntime {
	return &ScriptRuntime{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Invoke executes a JS export function from a bundled script file.
func (rt *ScriptRuntime) Invoke(ctx context.Context, vendorPath, scriptPath, exportName string, payload map[string]any) (any, error) {
	vm := goja.New()
	rt.installFetch(vm)
	rt.installConsole(vm, scriptPath)

	if vendorPath != "" {
		vendorSource, err := os.ReadFile(vendorPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read vendor bundle %s: %w", vendorPath, err)
		}
		if _, err := vm.RunString(string(vendorSource)); err != nil {
			return nil, fmt.Errorf("failed to evaluate vendor bundle %s: %w", vendorPath, err)
		}
	}

	source, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script %s: %w", scriptPath, err)
	}

	if _, err := vm.RunString(string(source)); err != nil {
		return nil, fmt.Errorf("failed to evaluate script %s: %w", scriptPath, err)
	}

	bundle := vm.Get("HyperterseBundle")
	if goja.IsUndefined(bundle) || goja.IsNull(bundle) {
		return nil, fmt.Errorf("script %s does not expose HyperterseBundle exports", scriptPath)
	}
	bundleObj := bundle.ToObject(vm)
	fnValue := bundleObj.Get(exportName)
	fn, ok := goja.AssertFunction(fnValue)
	if !ok {
		return nil, fmt.Errorf("script %s does not export function '%s'", scriptPath, exportName)
	}

	result, err := fn(goja.Undefined(), vm.ToValue(payload), vm.ToValue(map[string]any{
		"deadlineUnixMilli": deadlineUnixMilli(ctx),
	}))
	if err != nil {
		return nil, fmt.Errorf("script function '%s' failed: %w", exportName, err)
	}

	exported := result.Export()
	if promise, ok := exported.(*goja.Promise); ok {
		switch promise.State() {
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("script promise rejected: %s", formatJSValue(vm, promise.Result()))
		case goja.PromiseStateFulfilled:
			return promise.Result().Export(), nil
		default:
			return nil, fmt.Errorf("script promise is pending; async operation did not settle")
		}
	}
	return exported, nil
}

func (rt *ScriptRuntime) installConsole(vm *goja.Runtime, scriptPath string) {
	log := logger.New("script")
	scriptName := filepath.Base(scriptPath)

	formatArgs := func(args []goja.Value) string {
		if len(args) == 0 {
			return ""
		}
		parts := make([]string, 0, len(args))
		for _, arg := range args {
			switch {
			case goja.IsUndefined(arg):
				parts = append(parts, "undefined")
			case goja.IsNull(arg):
				parts = append(parts, "null")
			default:
				exported := arg.Export()
				switch v := exported.(type) {
				case string:
					parts = append(parts, v)
				default:
					b, err := json.Marshal(exported)
					if err != nil {
						parts = append(parts, fmt.Sprintf("%v", exported))
					} else {
						parts = append(parts, string(b))
					}
				}
			}
		}
		return strings.Join(parts, " ")
	}

	consoleFn := func(level string) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			msg := formatArgs(call.Arguments)
			line := fmt.Sprintf("%s: %s", scriptName, msg)
			switch level {
			case "error":
				log.Error(line)
			case "warn":
				log.Warn(line)
			case "info":
				log.Info(line)
			case "debug":
				log.Debug(line)
			default:
				// Keep console.log at DEBUG, but mirror to INFO when DEBUG is disabled
				// so developers still see script logs in terminal defaults.
				log.Debug(line)
				if logger.GetLogLevel() < logger.LogLevelDebug {
					log.Info(line)
				}
			}
			return goja.Undefined()
		}
	}

	_ = vm.Set("console", map[string]any{
		"log":   consoleFn("log"),
		"debug": consoleFn("debug"),
		"info":  consoleFn("info"),
		"warn":  consoleFn("warn"),
		"error": consoleFn("error"),
	})
}

func formatJSValue(vm *goja.Runtime, value goja.Value) string {
	switch {
	case goja.IsUndefined(value):
		return "undefined"
	case goja.IsNull(value):
		return "null"
	}

	obj := value.ToObject(vm)
	if obj != nil {
		if msg := obj.Get("message"); !goja.IsUndefined(msg) && !goja.IsNull(msg) {
			msgStr := msg.String()
			if stack := obj.Get("stack"); !goja.IsUndefined(stack) && !goja.IsNull(stack) {
				stackStr := stack.String()
				if stackStr != "" && stackStr != msgStr {
					return msgStr + "\n" + stackStr
				}
			}
			return msgStr
		}
	}

	exported := value.Export()
	if s, ok := exported.(string); ok {
		return s
	}
	b, err := json.Marshal(exported)
	if err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", exported)
}

func deadlineUnixMilli(ctx context.Context) int64 {
	d, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	return d.UnixMilli()
}

func (rt *ScriptRuntime) installFetch(vm *goja.Runtime) {
	_ = vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("fetch(url, options) requires url"))
		}
		url := call.Argument(0).String()
		method := http.MethodGet
		body := ""
		headers := map[string]string{}

		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) && !goja.IsNull(call.Argument(1)) {
			opts := call.Argument(1).ToObject(vm)
			if m := opts.Get("method"); !goja.IsUndefined(m) {
				method = m.String()
			}
			if b := opts.Get("body"); !goja.IsUndefined(b) && !goja.IsNull(b) {
				body = b.String()
			}
			if h := opts.Get("headers"); !goja.IsUndefined(h) && !goja.IsNull(h) {
				headerObj := h.ToObject(vm)
				for _, key := range headerObj.Keys() {
					headers[key] = headerObj.Get(key).String()
				}
			}
		}

		req, err := http.NewRequest(method, url, io.NopCloser(stringsReader(body)))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := rt.httpClient.Do(req)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		respText := string(respBytes)

		response := map[string]any{
			"status": resp.StatusCode,
			"ok":     resp.StatusCode >= 200 && resp.StatusCode < 300,
			"text": func() string {
				return respText
			},
			"json": func() map[string]any {
				var m map[string]any
				_ = json.Unmarshal(respBytes, &m)
				return m
			},
		}
		return vm.ToValue(response)
	})
}

type strReader struct {
	s string
	i int
}

func stringsReader(s string) *strReader { return &strReader{s: s} }

func (r *strReader) Read(p []byte) (int, error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}
