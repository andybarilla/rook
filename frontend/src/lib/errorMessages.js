const patterns = [
  {
    match: /is not a directory/,
    message: () => 'The selected path is not a valid directory.',
  },
  {
    match: /domain "([^"]+)" is already registered/,
    message: (m) => `A site with domain "${m[1]}" already exists.`,
  },
  {
    match: /domain "([^"]+)" not found/,
    message: (m) => `Could not find site "${m[1]}".`,
  },
];

export function friendlyError(raw) {
  if (!raw) return raw;
  for (const { match, message } of patterns) {
    const m = raw.match(match);
    if (m) return message(m);
  }
  return raw;
}
