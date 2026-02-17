var HyperterseBundle = (() => {
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

  // demo/app/routes/get-weather/demo/app/routes/get-weather/weather-handler.ts
  var weather_handler_exports = {};
  __export(weather_handler_exports, {
    handler: () => handler
  });
  var __ht_vendor_mod_1 = (globalThis.__hyperterse_vendor || {})["dayjs"];
  if (!__ht_vendor_mod_1) {
    throw new Error("Missing vendor module: dayjs");
  }
  var dayjs = __ht_vendor_mod_1.default ?? __ht_vendor_mod_1;
  async function handler(payload) {
    const city = payload?.inputs?.city ?? "Bengaluru";
    const units = payload?.inputs?.units ?? "metric";
    const sampleTemp = units === "imperial" ? 77 : 25;
    const sampleWind = units === "imperial" ? "8 mph" : "13 km/h";
    return [
      {
        city,
        units,
        observed_at: dayjs().toISOString(),
        weather: "Partly cloudy",
        temperature: sampleTemp,
        wind: sampleWind,
        source: "demo-handler",
        route: payload?.route ?? "unknown"
      }
    ];
  }
  return __toCommonJS(weather_handler_exports);
})();
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsiLi4vLi4vLi4vLi4vYXBwL3JvdXRlcy9nZXQtd2VhdGhlci9kZW1vL2FwcC9yb3V0ZXMvZ2V0LXdlYXRoZXIvd2VhdGhlci1oYW5kbGVyLnRzIl0sCiAgInNvdXJjZXNDb250ZW50IjogWyJjb25zdCBfX2h0X3ZlbmRvcl9tb2RfMSA9IChnbG9iYWxUaGlzLl9faHlwZXJ0ZXJzZV92ZW5kb3IgfHwge30pW1wiZGF5anNcIl07IGlmICghX19odF92ZW5kb3JfbW9kXzEpIHsgdGhyb3cgbmV3IEVycm9yKFwiTWlzc2luZyB2ZW5kb3IgbW9kdWxlOiBkYXlqc1wiKTsgfVxuY29uc3QgZGF5anMgPSAoX19odF92ZW5kb3JfbW9kXzEuZGVmYXVsdCA/PyBfX2h0X3ZlbmRvcl9tb2RfMSk7XG5cbnR5cGUgSW5wdXRzID0ge1xuICBjaXR5Pzogc3RyaW5nO1xuICB1bml0cz86IFwibWV0cmljXCIgfCBcImltcGVyaWFsXCI7XG59O1xuXG5leHBvcnQgYXN5bmMgZnVuY3Rpb24gaGFuZGxlcihwYXlsb2FkOiB7IGlucHV0cz86IElucHV0czsgcm91dGU/OiBzdHJpbmcgfSkge1xuICBjb25zdCBjaXR5ID0gcGF5bG9hZD8uaW5wdXRzPy5jaXR5ID8/IFwiQmVuZ2FsdXJ1XCI7XG4gIGNvbnN0IHVuaXRzID0gcGF5bG9hZD8uaW5wdXRzPy51bml0cyA/PyBcIm1ldHJpY1wiO1xuXG4gIGNvbnN0IHNhbXBsZVRlbXAgPSB1bml0cyA9PT0gXCJpbXBlcmlhbFwiID8gNzcgOiAyNTtcbiAgY29uc3Qgc2FtcGxlV2luZCA9IHVuaXRzID09PSBcImltcGVyaWFsXCIgPyBcIjggbXBoXCIgOiBcIjEzIGttL2hcIjtcblxuICByZXR1cm4gW1xuICAgIHtcbiAgICAgIGNpdHksXG4gICAgICB1bml0cyxcbiAgICAgIG9ic2VydmVkX2F0OiBkYXlqcygpLnRvSVNPU3RyaW5nKCksXG4gICAgICB3ZWF0aGVyOiBcIlBhcnRseSBjbG91ZHlcIixcbiAgICAgIHRlbXBlcmF0dXJlOiBzYW1wbGVUZW1wLFxuICAgICAgd2luZDogc2FtcGxlV2luZCxcbiAgICAgIHNvdXJjZTogXCJkZW1vLWhhbmRsZXJcIixcbiAgICAgIHJvdXRlOiBwYXlsb2FkPy5yb3V0ZSA/PyBcInVua25vd25cIlxuICAgIH1cbiAgXTtcbn1cbiJdLAogICJtYXBwaW5ncyI6ICI7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O0FBQUE7QUFBQTtBQUFBO0FBQUE7QUFBQSxNQUFNLHFCQUFxQixXQUFXLHVCQUF1QixDQUFDLEdBQUcsT0FBTztBQUFHLE1BQUksQ0FBQyxtQkFBbUI7QUFBRSxVQUFNLElBQUksTUFBTSw4QkFBOEI7QUFBQSxFQUFHO0FBQ3RKLE1BQU0sUUFBUyxrQkFBa0IsV0FBVztBQU81QyxpQkFBc0IsUUFBUSxTQUE4QztBQUMxRSxVQUFNLE9BQU8sU0FBUyxRQUFRLFFBQVE7QUFDdEMsVUFBTSxRQUFRLFNBQVMsUUFBUSxTQUFTO0FBRXhDLFVBQU0sYUFBYSxVQUFVLGFBQWEsS0FBSztBQUMvQyxVQUFNLGFBQWEsVUFBVSxhQUFhLFVBQVU7QUFFcEQsV0FBTztBQUFBLE1BQ0w7QUFBQSxRQUNFO0FBQUEsUUFDQTtBQUFBLFFBQ0EsYUFBYSxNQUFNLEVBQUUsWUFBWTtBQUFBLFFBQ2pDLFNBQVM7QUFBQSxRQUNULGFBQWE7QUFBQSxRQUNiLE1BQU07QUFBQSxRQUNOLFFBQVE7QUFBQSxRQUNSLE9BQU8sU0FBUyxTQUFTO0FBQUEsTUFDM0I7QUFBQSxJQUNGO0FBQUEsRUFDRjsiLAogICJuYW1lcyI6IFtdCn0K
