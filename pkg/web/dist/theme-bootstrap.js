(() => {
  const themes = ['light', 'dark', 'bw'];
  const accents = ['sky', 'violet', 'emerald', 'amber', 'rose', 'mono'];
  const storedTheme = localStorage.getItem('modbridge_theme');
  const legacyTheme = localStorage.getItem('theme');
  const theme = themes.includes(storedTheme)
    ? storedTheme
    : themes.includes(legacyTheme)
      ? legacyTheme
      : 'light';
  const storedAccent = localStorage.getItem('modbridge_accent');
  const storedDensity = localStorage.getItem('modbridge_density');
  const reducedMotion =
    localStorage.getItem('modbridge_reduced_motion') === 'true' ||
    window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  const html = document.documentElement;

  html.classList.toggle('light', theme === 'light');
  html.classList.toggle('dark', theme === 'dark' || theme === 'bw');
  html.classList.toggle('bw', theme === 'bw');
  html.classList.toggle('reduced-motion', reducedMotion);
  html.dataset.accent = accents.includes(storedAccent) ? storedAccent : 'mono';
  html.dataset.density = storedDensity === 'compact' ? 'compact' : 'comfortable';
})();
