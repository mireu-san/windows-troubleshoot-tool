export function languageFor(locale = '') {
  const base = locale.toLowerCase().split(/[-_]/)[0];
  return ['ko', 'ja'].includes(base) ? base : 'en';
}
export function resolveLanguage(preference, systemLanguage) {
  return ['ko', 'en', 'ja'].includes(preference) ? preference : languageFor(systemLanguage);
}
export function createTranslator(messages) {
  const escapeRegExp = text => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const keys = Object.keys(messages).sort((a, b) => b.length - a.length);
  const literalPattern = new RegExp(keys.filter(key => !/%[dsw]/.test(key)).map(escapeRegExp).join('|'), 'g');
  const formatted = keys.filter(key => /%[dsw]/.test(key)).map(key => ({
    key,
    pattern: new RegExp(key.split(/%[dsw]/).map(escapeRegExp).join('(.+?)') + '(?=$|\\n)'),
  }));
  function translateText(source, language) {
    if (source == null) return '';
    source = String(source);
    if (language === 'ko' || !['en', 'ja'].includes(language)) return source;
    if (messages[source]) return messages[source][language];
    // App-authored formatted status/errors may contain command names and durations.
    // Unknown operating-system text is retained verbatim.
    let translated = source;
    for (const { key, pattern } of formatted) {
      translated = translated.replace(pattern, (...match) => {
        let index = 1;
        return messages[key][language].replace(/%[dsw]/g, () => translateText(match[index++], language));
      });
    }
    return translated.replace(literalPattern, key => messages[key][language]);
  }

  return translateText;
}
