import { Type } from "typebox";

/** Shared session parameter -- optional on every tool. */
export const SessionParam = Type.Optional(Type.String({
  description: "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). " +
    "Omit to use the default session.",
}));
