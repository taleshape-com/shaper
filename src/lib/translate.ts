
const translations: Record<string, Record<string, string>> = {
  'No data available': {
    de: 'Keine Daten verfügbar'
  },
  'Apply': {
    de: 'Bestätigen'
  },
  'Cancel': {
    de: 'Abbrechen'
  },
  'Select date': {
    'de': 'Datum wählen'
  },
  'Select date range': {
    de: 'Zeitraum wählen'
  },
  "Today": {
    de: 'Heute'
  },
  "Last 7 days": {
    de: 'Letzte 7 Tage'
  },
  "Last 30 days": {
    de: 'Letzte 30 Tage'
  },
  "Last 3 months": {
    de: 'Letzte 3 Monate'
  },
  "Last 6 months": {
    de: 'Letzte 6 Monate'
  },
  "Month to date": {
    de: 'Monat bis heute'
  },
  "Year to date": {
    de: 'Jahr bis heute'
  },
}

export function translate(s: string) {
  const available = translations[s] ?? {}
  for (const lang of navigator.languages) {
    if (lang === 'en' || lang === 'en-US') {
      return s
    }
    const t = available[lang]
    if (t) {
      return t
    }
  }
  return s
}

