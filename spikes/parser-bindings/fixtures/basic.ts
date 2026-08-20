import type { RenderOptions } from "example-types";
import { escape as escapeHTML } from "lodash";

export function render(
    value: string,
    options?: RenderOptions
): string {
    void options;
    return escapeHTML(value);
}
