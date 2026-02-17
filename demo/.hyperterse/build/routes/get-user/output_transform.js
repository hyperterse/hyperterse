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

  // demo/app/routes/get-user/demo/app/routes/get-user/user-data-mapper.ts
  var user_data_mapper_exports = {};
  __export(user_data_mapper_exports, {
    outputTransform: () => outputTransform
  });
  var __ht_vendor_mod_1 = (globalThis.__hyperterse_vendor || {})["dayjs"];
  if (!__ht_vendor_mod_1) {
    throw new Error("Missing vendor module: dayjs");
  }
  var dayjs = __ht_vendor_mod_1.default ?? __ht_vendor_mod_1;
  var __ht_vendor_mod_2 = (globalThis.__hyperterse_vendor || {})["uuid"];
  if (!__ht_vendor_mod_2) {
    throw new Error("Missing vendor module: uuid");
  }
  var uuidv4 = __ht_vendor_mod_2["v4"];
  async function outputTransform(payload) {
    const rows = payload?.results ?? [];
    console.log("outputTransform", rows);
    return rows.map((row) => ({
      trace_id: uuidv4(),
      id: row.id,
      name: row.name,
      email: row.email,
      created_at_iso: row.created_at ? dayjs(String(row.created_at)).toISOString() : null
    }));
  }
  return __toCommonJS(user_data_mapper_exports);
})();
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsiLi4vLi4vLi4vLi4vYXBwL3JvdXRlcy9nZXQtdXNlci9kZW1vL2FwcC9yb3V0ZXMvZ2V0LXVzZXIvdXNlci1kYXRhLW1hcHBlci50cyJdLAogICJzb3VyY2VzQ29udGVudCI6IFsiY29uc3QgX19odF92ZW5kb3JfbW9kXzEgPSAoZ2xvYmFsVGhpcy5fX2h5cGVydGVyc2VfdmVuZG9yIHx8IHt9KVtcImRheWpzXCJdOyBpZiAoIV9faHRfdmVuZG9yX21vZF8xKSB7IHRocm93IG5ldyBFcnJvcihcIk1pc3NpbmcgdmVuZG9yIG1vZHVsZTogZGF5anNcIik7IH1cbmNvbnN0IGRheWpzID0gKF9faHRfdmVuZG9yX21vZF8xLmRlZmF1bHQgPz8gX19odF92ZW5kb3JfbW9kXzEpO1xuY29uc3QgX19odF92ZW5kb3JfbW9kXzIgPSAoZ2xvYmFsVGhpcy5fX2h5cGVydGVyc2VfdmVuZG9yIHx8IHt9KVtcInV1aWRcIl07IGlmICghX19odF92ZW5kb3JfbW9kXzIpIHsgdGhyb3cgbmV3IEVycm9yKFwiTWlzc2luZyB2ZW5kb3IgbW9kdWxlOiB1dWlkXCIpOyB9XG5jb25zdCB1dWlkdjQgPSBfX2h0X3ZlbmRvcl9tb2RfMltcInY0XCJdO1xuXG50eXBlIFJvdyA9IFJlY29yZDxzdHJpbmcsIHVua25vd24+O1xuXG5leHBvcnQgYXN5bmMgZnVuY3Rpb24gb3V0cHV0VHJhbnNmb3JtKHBheWxvYWQ6IHsgcmVzdWx0cz86IFJvd1tdIH0pIHtcbiAgY29uc3Qgcm93cyA9IHBheWxvYWQ/LnJlc3VsdHMgPz8gW107XG4gIGNvbnNvbGUubG9nKFwib3V0cHV0VHJhbnNmb3JtXCIsIHJvd3MpO1xuICByZXR1cm4gcm93cy5tYXAoKHJvdykgPT4gKHtcbiAgICB0cmFjZV9pZDogdXVpZHY0KCksXG4gICAgaWQ6IHJvdy5pZCxcbiAgICBuYW1lOiByb3cubmFtZSxcbiAgICBlbWFpbDogcm93LmVtYWlsLFxuICAgIGNyZWF0ZWRfYXRfaXNvOiByb3cuY3JlYXRlZF9hdCA/IGRheWpzKFN0cmluZyhyb3cuY3JlYXRlZF9hdCkpLnRvSVNPU3RyaW5nKCkgOiBudWxsLFxuICB9KSk7XG59XG4iXSwKICAibWFwcGluZ3MiOiAiOzs7Ozs7Ozs7Ozs7Ozs7Ozs7OztBQUFBO0FBQUE7QUFBQTtBQUFBO0FBQUEsTUFBTSxxQkFBcUIsV0FBVyx1QkFBdUIsQ0FBQyxHQUFHLE9BQU87QUFBRyxNQUFJLENBQUMsbUJBQW1CO0FBQUUsVUFBTSxJQUFJLE1BQU0sOEJBQThCO0FBQUEsRUFBRztBQUN0SixNQUFNLFFBQVMsa0JBQWtCLFdBQVc7QUFDNUMsTUFBTSxxQkFBcUIsV0FBVyx1QkFBdUIsQ0FBQyxHQUFHLE1BQU07QUFBRyxNQUFJLENBQUMsbUJBQW1CO0FBQUUsVUFBTSxJQUFJLE1BQU0sNkJBQTZCO0FBQUEsRUFBRztBQUNwSixNQUFNLFNBQVMsa0JBQWtCLElBQUk7QUFJckMsaUJBQXNCLGdCQUFnQixTQUE4QjtBQUNsRSxVQUFNLE9BQU8sU0FBUyxXQUFXLENBQUM7QUFDbEMsWUFBUSxJQUFJLG1CQUFtQixJQUFJO0FBQ25DLFdBQU8sS0FBSyxJQUFJLENBQUMsU0FBUztBQUFBLE1BQ3hCLFVBQVUsT0FBTztBQUFBLE1BQ2pCLElBQUksSUFBSTtBQUFBLE1BQ1IsTUFBTSxJQUFJO0FBQUEsTUFDVixPQUFPLElBQUk7QUFBQSxNQUNYLGdCQUFnQixJQUFJLGFBQWEsTUFBTSxPQUFPLElBQUksVUFBVSxDQUFDLEVBQUUsWUFBWSxJQUFJO0FBQUEsSUFDakYsRUFBRTtBQUFBLEVBQ0o7IiwKICAibmFtZXMiOiBbXQp9Cg==
