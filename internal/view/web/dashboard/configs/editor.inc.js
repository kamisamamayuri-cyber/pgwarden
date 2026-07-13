window.alpineConfigEditor = function (textareaId) {
  return {
    status: null, // null | 'ok' | 'error'
    errorMsg: "",

    init() {
      const ta = document.getElementById(textareaId);
      if (!ta) return;

      const validate = window.debounce(() => {
        const result = window.yamlLint(ta.value);
        this.status = result.ok ? "ok" : "error";
        this.errorMsg = result.error || "";
      }, 400);

      ta.addEventListener("input", validate);
      // Run once on load to show status for existing content.
      validate();
    },
  };
};
