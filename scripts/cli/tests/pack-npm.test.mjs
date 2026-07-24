import assert from "node:assert/strict";
import test from "node:test";

import {
  validatePackFiles,
  validatePublishManifest,
} from "../pack-npm.mjs";

test("accepts the public npm package file whitelist", () => {
  assert.doesNotThrow(() => validatePackFiles([
    { path: "package/package.json" },
    { path: "package/README.md" },
    { path: "package/LICENSE" },
    { path: "package/bin/luna.js" },
    { path: "package/dist/entry.js" },
  ]));
});

test("rejects secrets, source maps and tests from the tarball", () => {
  assert.throws(
    () => validatePackFiles([
      { path: "package/dist/entry.js.map" },
      { path: "package/.env.production" },
      { path: "package/tests/auth.test.js" },
    ]),
    /disallowed files/,
  );
});

test("rejects lifecycle scripts and unresolved workspace dependencies", () => {
  assert.throws(
    () => validatePublishManifest({ scripts: { postinstall: "node setup.js" } }),
    /postinstall/,
  );
  assert.throws(
    () => validatePublishManifest({
      dependencies: { "@luna-devops/api-client": "workspace:*" },
    }),
    /workspace protocol/,
  );
});
