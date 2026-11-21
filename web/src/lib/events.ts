/**
 * Custom DOM events that let independent components talk to each other
 * without importing one another.
 */

/** Fired on document when a state is picked on the map. */
export const MAP_STATE_SELECT = "map:state-select";

export interface MapStateSelectDetail {
  /** OCHA P-code of the state, e.g. "VE14". */
  code: string;
}
