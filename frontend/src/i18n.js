import messages from './locales/messages.json';
import { createTranslator, languageFor, resolveLanguage } from './localization.js';
const translateText = createTranslator(messages);

let settings = { preference: 'auto', language: 'en' };
export const t = (source, values = {}) => translateText(source, settings.language)
  .replace(/\{(\w+)\}/g, (match, name) => values[name] == null ? match : values[name]);
const bridge = () => globalThis.window?.go?.main?.App;
const storageKey = 'windows-repair-language';
const sourceTexts = new WeakMap();
const sourceAttributes = new WeakMap();
const skipped = 'pre,script,style,[data-i18n-skip]';
let observer;

// Keep the original text with each DOM node so changing language never rebuilds
// controls, loses focus, clears answers, or interrupts a running operation.
function localizeTextNode(node) {
  if (node.parentElement?.closest(skipped)) return;
  let record = sourceTexts.get(node);
  if (!record || node.data !== record.rendered) record = { source: node.data };
  record.rendered = t(record.source);
  sourceTexts.set(node, record);
  if (node.data !== record.rendered) node.data = record.rendered;
}
function localizeElement(element) {
  if (element.closest(skipped)) return;
  const records = sourceAttributes.get(element) || {};
  for (const name of ['title', 'aria-label', 'placeholder']) {
    if (!element.hasAttribute(name)) continue;
    const current = element.getAttribute(name);
    let record = records[name];
    if (!record || current !== record.rendered) record = { source: current };
    record.rendered = t(record.source);
    records[name] = record;
    if (current !== record.rendered) element.setAttribute(name, record.rendered);
  }
  sourceAttributes.set(element, records);
}
function localizeTree(root) {
  if (root.nodeType === Node.TEXT_NODE) return localizeTextNode(root);
  if (root.nodeType !== Node.ELEMENT_NODE || root.closest(skipped)) return;
  localizeElement(root);
  for (const child of root.childNodes) localizeTree(child);
}
function watch() {
  observer.observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ['title', 'aria-label', 'placeholder'] });
}
function refresh() {
  observer.disconnect();
  localizeTree(document.body);
  document.documentElement.lang = settings.language;
  document.title = t('Windows 시스템 복구 도우미');
  document.querySelectorAll('[data-language-picker]').forEach(select => { select.value = settings.preference; });
  watch();
  window.dispatchEvent(new CustomEvent('language:changed'));
}
export async function initI18n() {
  const system = languageFor(navigator.language);
  try {
    settings = bridge()?.GetLanguageSettings
      ? await bridge().GetLanguageSettings()
      : { preference: localStorage.getItem(storageKey) || 'auto', language: system };
  } catch { settings = { preference: 'auto', language: system }; }
  settings.language = resolveLanguage(settings.preference, settings.language);
  observer = new MutationObserver(records => {
    observer.disconnect();
    for (const record of records) {
      if (record.type === 'childList') record.addedNodes.forEach(localizeTree);
      else if (record.type === 'characterData') localizeTextNode(record.target);
      else localizeElement(record.target);
    }
    watch();
  });
  refresh();
}
export function mountLanguagePicker(parent) {
  const label = document.createElement('label');
  label.className = 'language-picker';
  const caption = document.createElement('span');
  caption.textContent = '언어';
  const select = document.createElement('select');
  select.dataset.languagePicker = '';
  select.setAttribute('aria-label', '언어');
  for (const [value, text] of [['auto', '시스템 설정 따르기'], ['ko', '한국어'], ['en', 'English'], ['ja', '日本語']]) {
    const option = new Option(text, value);
    if (value !== 'auto') option.dataset.i18nSkip = '';
    select.add(option);
  }
  select.value = settings.preference;
  label.append(caption, select);
  parent.append(label);
  select.addEventListener('change', async () => {
    const preference = select.value;
    document.querySelectorAll('[data-language-picker]').forEach(el => { el.disabled = true; });
    try {
      if (bridge()?.SetLanguage) settings = await bridge().SetLanguage(preference);
      else {
        localStorage.setItem(storageKey, preference);
        settings = { preference, language: resolveLanguage(preference, navigator.language) };
      }
      refresh();
    } catch {
      select.value = settings.preference;
      window.alert(t('언어 설정을 저장하지 못했습니다. 다시 시도해 주세요.'));
    } finally {
      document.querySelectorAll('[data-language-picker]').forEach(el => { el.disabled = false; });
    }
  });
}
