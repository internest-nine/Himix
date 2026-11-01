// Kept separate from the navigation component so pages can import the page id
// union without pulling in the dock's icon bundle.
export type AppPage =
  | "tools"
  | "home"
  | "routing"
  | "health"
  | "connections"
  | "settings"
  | "blocked-domains"
  | "about"
  | "appearance";
