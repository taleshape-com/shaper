
const translations: Record<string, Record<string, string>> = {
  'loading': {
    de: 'Einen Moment'
  },
  'Overview': {
    de: 'Übersicht'
  },
  'Name': {
    de: 'Name'
  },
  'Created': {
    de: 'Erstellt'
  },
  'Updated': {
    de: 'Aktualisiert'
  },
  'Actions': {
    de: 'Aktionen'
  },
  'Edit': {
    de: 'Bearbeiten'
  },
  'Create': {
    de: 'Erstellen'
  },
  'Creating': {
    de: 'Erstellen'
  },
  'Delete': {
    de: 'Löschen'
  },
  'Delete Dashboard': {
    de: 'Dashboard löschen'
  },
  'New': {
    de: 'Neu'
  },
  'Save': {
    de: 'Speichern'
  },
  'Saving': {
    de: 'Speichern'
  },
  'Edit Dashboard': {
    de: 'Dashboard bearbeiten'
  },
  'View Dashboard': {
    de: 'Dashboard anzeigen'
  },
  'Enter a name for the dashboard': {
    de: 'Geben Sie einen Namen für das Dashboard ein'
  },
  'Are you sure you want to delete the dashboard "%%"?': {
    de: 'Sind Sie sicher, dass Sie das Dashboard "%%" löschen möchten?'
  },
  'There are unsaved previous edits. Do you want to restore them?': {
    de: 'Es gibt ungespeicherte Änderungen. Möchten Sie diese wiederherstellen?'
  },
  'No data available': {
    de: 'Keine Daten verfügbar'
  },
  'Nothing to show yet': {
    de: 'Noch nichts zu zeigen'
  },
  'Variables': {
    de: 'Parameter'
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

