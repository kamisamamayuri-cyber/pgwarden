import * as YAML from "yaml";

export function initYamlValidator() {
  window.yamlLint = function (content) {
    if (!content || !content.trim()) {
      return { ok: false, error: "Файл пустой" };
    }
    try {
      YAML.parse(content, { strict: true });
      return { ok: true, error: null };
    } catch (e) {
      return { ok: false, error: e.message };
    }
  };
}
