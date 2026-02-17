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

  // demo/app/routes/get-user/user-input-validator.ts
  var user_input_validator_exports = {};
  __export(user_input_validator_exports, {
    inputTransform: () => inputTransform
  });
  async function inputTransform(payload) {
    console.log("inputTransform,,,,,", payload);
    const inputs = payload?.inputs ?? {};
    const rawUserId = inputs.userId;
    if (rawUserId === void 0 || rawUserId === null) {
      throw new Error("user_id is required");
    }
    const userId = Number(rawUserId);
    if (!Number.isFinite(userId) || userId <= 0 || !Number.isInteger(userId)) {
      throw new Error("user_id must be a positive integer");
    }
    return {
      ...inputs,
      userId
    };
  }
  return __toCommonJS(user_input_validator_exports);
})();
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsiLi4vLi4vLi4vLi4vYXBwL3JvdXRlcy9nZXQtdXNlci91c2VyLWlucHV0LXZhbGlkYXRvci50cyJdLAogICJzb3VyY2VzQ29udGVudCI6IFsiZXhwb3J0IGFzeW5jIGZ1bmN0aW9uIGlucHV0VHJhbnNmb3JtKHBheWxvYWQ6IHsgaW5wdXRzPzogUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfSkge1xuICBjb25zb2xlLmxvZyhcImlucHV0VHJhbnNmb3JtLCwsLCxcIiwgcGF5bG9hZCk7XG4gIGNvbnN0IGlucHV0cyA9IHBheWxvYWQ/LmlucHV0cyA/PyB7fTtcbiAgY29uc3QgcmF3VXNlcklkID0gaW5wdXRzLnVzZXJJZDtcblxuICBpZiAocmF3VXNlcklkID09PSB1bmRlZmluZWQgfHwgcmF3VXNlcklkID09PSBudWxsKSB7XG4gICAgdGhyb3cgbmV3IEVycm9yKFwidXNlcl9pZCBpcyByZXF1aXJlZFwiKTtcbiAgfVxuXG4gIGNvbnN0IHVzZXJJZCA9IE51bWJlcihyYXdVc2VySWQpO1xuICBpZiAoIU51bWJlci5pc0Zpbml0ZSh1c2VySWQpIHx8IHVzZXJJZCA8PSAwIHx8ICFOdW1iZXIuaXNJbnRlZ2VyKHVzZXJJZCkpIHtcbiAgICB0aHJvdyBuZXcgRXJyb3IoXCJ1c2VyX2lkIG11c3QgYmUgYSBwb3NpdGl2ZSBpbnRlZ2VyXCIpO1xuICB9XG5cbiAgcmV0dXJuIHtcbiAgICAuLi5pbnB1dHMsXG4gICAgdXNlcklkLFxuICB9O1xufVxuIl0sCiAgIm1hcHBpbmdzIjogIjs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7QUFBQTtBQUFBO0FBQUE7QUFBQTtBQUFBLGlCQUFzQixlQUFlLFNBQStDO0FBQ2xGLFlBQVEsSUFBSSx1QkFBdUIsT0FBTztBQUMxQyxVQUFNLFNBQVMsU0FBUyxVQUFVLENBQUM7QUFDbkMsVUFBTSxZQUFZLE9BQU87QUFFekIsUUFBSSxjQUFjLFVBQWEsY0FBYyxNQUFNO0FBQ2pELFlBQU0sSUFBSSxNQUFNLHFCQUFxQjtBQUFBLElBQ3ZDO0FBRUEsVUFBTSxTQUFTLE9BQU8sU0FBUztBQUMvQixRQUFJLENBQUMsT0FBTyxTQUFTLE1BQU0sS0FBSyxVQUFVLEtBQUssQ0FBQyxPQUFPLFVBQVUsTUFBTSxHQUFHO0FBQ3hFLFlBQU0sSUFBSSxNQUFNLG9DQUFvQztBQUFBLElBQ3REO0FBRUEsV0FBTztBQUFBLE1BQ0wsR0FBRztBQUFBLE1BQ0g7QUFBQSxJQUNGO0FBQUEsRUFDRjsiLAogICJuYW1lcyI6IFtdCn0K
