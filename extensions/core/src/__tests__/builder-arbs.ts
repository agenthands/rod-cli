import fc from "fast-check";
import * as b from "../argv";

/** Hostile values a model could put in any string field (gap-plan ext:EXT-1). */
export const HOSTILE = ["-s=victim", "--session=x", "--json", "--cdp=ws://x", "--out", "--"];

/** A model-controlled string field. */
export const modelString = fc.oneof(fc.string({ minLength: 1 }), fc.constantFrom(...HOSTILE));

/** The `out` path captureScreenshot generates itself; the model never sets it. */
export const SCREENSHOT_OUT = "/tmp/rod-cli-out.png";

const opt = <T>(a: fc.Arbitrary<T>) => fc.option(a, { nil: undefined });
const s = modelString;

/** One call of any argv builder, with model-drawn fields and the given session. */
export function builderCall(session: fc.Arbitrary<string | undefined>) {
  const call = <M extends object>(name: string, model: fc.Arbitrary<M>, build: (m: M & { session?: string }) => string[]) =>
    fc.tuple(model, session).map(([m, sess]) => {
      const full = { ...m, session: sess };
      return { name, model: full, argv: build(full) };
    });
  return fc.oneof(
    call("goto", fc.record({ url: s }), b.buildGotoArgs),
    call("snapshot", fc.record({ selector: opt(s), interactive: opt(fc.boolean()) }), b.buildSnapshotArgs),
    call("click", fc.record({ selector: s, doubleClick: opt(fc.boolean()) }), b.buildClickArgs),
    call("type", fc.record({ selector: s, text: s }), b.buildTypeArgs),
    call("eval", fc.record({ expression: s }), b.buildEvalArgs),
    call("screenshot", fc.record({ selector: opt(s), fullPage: opt(fc.boolean()) }), (m) =>
      b.buildScreenshotArgs({ ...m, out: SCREENSHOT_OUT })),
    call("scroll", fc.oneof(
      fc.record({ direction: fc.constant("down"), selector: opt(s) }),
      fc.record({ direction: fc.constantFrom("up", "down", "left", "right"), distance: fc.nat(), selector: opt(s) }),
    ), b.buildScrollArgs),
    call("tabs", fc.oneof(
      fc.record({ action: fc.constant("list") }),
      fc.record({ action: fc.constant("new"), url: s }),
      fc.record({ action: fc.constantFrom("close", "select"), index: fc.nat() }),
    ), b.buildTabsArgs),
    call("navigate", fc.record({ action: fc.constantFrom("reload", "back", "forward") }), b.buildNavigateArgs),
    call("cookies", fc.oneof(
      fc.record({ action: fc.constantFrom("get", "clear") }),
      fc.record({ action: fc.constant("set"), name: s, value: s }),
      fc.record({ action: fc.constant("delete"), name: s }),
    ), b.buildCookiesArgs),
    call("storage", fc.oneof(
      fc.record({ action: fc.constantFrom("get", "clear"), storageType: fc.constantFrom("local", "session") }),
      fc.record({ action: fc.constant("get"), storageType: fc.constantFrom("local", "session"), key: s }),
      fc.record({ action: fc.constant("set"), storageType: fc.constantFrom("local", "session"), key: s, value: s }),
      fc.record({ action: fc.constant("delete"), storageType: fc.constantFrom("local", "session"), key: s }),
    ), b.buildStorageArgs),
    call("fill", fc.record({ selector: s, text: s, submit: opt(fc.boolean()) }), b.buildFillFormArgs),
    call("press", fc.record({ key: s }), b.buildPressArgs),
    call("select", fc.record({ selector: s, values: fc.array(s, { maxLength: 3 }) }), b.buildSelectArgs),
    call("hover", fc.record({ selector: s }), b.buildHoverArgs),
  );
}
