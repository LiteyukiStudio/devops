import { describe, expect, it } from "vitest";
import {
  renderFieldView,
  renderNames,
  renderTable,
  renderYaml,
} from "../../src/output/index.js";

describe("human output rendering", () => {
  it("renders aligned tables with localized headers", () => {
    const rendered = renderTable(
      [
        { name: "应用", status: "ready" },
        { name: "api", status: "pending" },
      ],
      {
        columns: [
          { key: "name", header: "名称" },
          { key: "status", header: "状态" },
        ],
      },
    );
    expect(rendered.split("\n")).toHaveLength(3);
    expect(rendered).toContain("名称");
    expect(rendered).toContain("pending");
  });

  it("renders field, YAML and name views with redaction", () => {
    expect(renderFieldView({ name: "demo", password: "secret" }))
      .toContain("password  [REDACTED]");
    expect(renderYaml({ name: "demo", token: "secret" })).toContain("token: \"[REDACTED]\"");
    expect(renderNames([{ id: "id_1" }, { name: "demo" }])).toBe("id_1\ndemo");
  });
});
