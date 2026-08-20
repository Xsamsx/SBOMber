import { template as compile } from "lodash";

const path = require("node:path");

export function render(name) {
    const output = compile("Hello <%= name %>")({ name });
    return path.basename(output);
}
