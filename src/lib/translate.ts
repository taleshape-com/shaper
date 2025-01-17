const translations: Record<string, Record<string, string>> = {
  Error: {
    de: "Fehler",
  },
  "An error occurred": {
    de: "Ein Fehler ist aufgetreten",
  },
  Logout: {
    de: "Abmelden",
  },
  Admin: {
    de: "Admin",
  },
  loading: {
    de: "Einen Moment",
  },
  Overview: {
    de: "Übersicht",
  },
  Name: {
    de: "Name",
  },
  Created: {
    de: "Erstellt",
  },
  Updated: {
    de: "Aktualisiert",
  },
  Actions: {
    de: "Aktionen",
  },
  Edit: {
    de: "Bearbeiten",
  },
  Create: {
    de: "Erstellen",
  },
  Creating: {
    de: "Erstellen",
  },
  Delete: {
    de: "Löschen",
  },
  "Delete Dashboard": {
    de: "Dashboard löschen",
  },
  New: {
    de: "Neu",
  },
  Save: {
    de: "Speichern",
  },
  Saving: {
    de: "Speichern",
  },
  "Edit Dashboard": {
    de: "Dashboard bearbeiten",
  },
  "View Dashboard": {
    de: "Dashboard anzeigen",
  },
  "Loading preview": {
    de: "Vorschau laden",
  },
  "Enter a name for the dashboard": {
    de: "Geben Sie einen Namen für das Dashboard ein",
  },
  'Are you sure you want to delete the dashboard "%%"?': {
    de: 'Sind Sie sicher, dass Sie das Dashboard "%%" löschen möchten?',
  },
  "There are unsaved previous edits. Do you want to restore them?": {
    de: "Es gibt ungespeicherte Änderungen. Möchten Sie diese wiederherstellen?",
  },
  "No data available": {
    de: "Keine Daten verfügbar",
  },
  "Nothing to show yet": {
    de: "Noch nichts zu zeigen",
  },
  Variables: {
    de: "Parameter",
  },
  Apply: {
    de: "Bestätigen",
  },
  Cancel: {
    de: "Abbrechen",
  },
  "Select date": {
    de: "Datum wählen",
  },
  "Select date range": {
    de: "Zeitraum wählen",
  },
  Today: {
    de: "Heute",
  },
  "Last 7 days": {
    de: "Letzte 7 Tage",
  },
  "Last 30 days": {
    de: "Letzte 30 Tage",
  },
  "Last 3 months": {
    de: "Letzte 3 Monate",
  },
  "Last 6 months": {
    de: "Letzte 6 Monate",
  },
  "Month to date": {
    de: "Monat bis heute",
  },
  "Year to date": {
    de: "Jahr bis heute",
  },
  "Security Settings": {
    de: "Sicherheitseinstellungen",
  },
  "Reset the JWT secret to invalidate all existing tokens.": {
    de: "Setzen Sie das JWT-Secret zurück, um alle vorhandenen Token ungültig zu machen.",
  },
  "JWT secret reset successfully": {
    de: "JWT-Secret erfolgreich zurückgesetzt",
  },
  "Resetting...": {
    de: "Zurücksetzen...",
  },
  "Reset JWT Secret": {
    de: "JWT-Secret zurücksetzen",
  },
  "API Keys": {
    de: "API-Keys"
  },
  "Loading API keys...": {
    de: "API-Keys werden geladen..."
  },
  "No API keys found": {
    de: "Keine API-Keys gefunden"
  },
  "Create New API Key": {
    de: "Neuen API-Key erstellen"
  },
  "Key:": {
    de: "Schlüssel:"
  },
  "API key copied to clipboard": {
    de: "API-Key wurde in die Zwischenablage kopiert"
  },
  "Make sure to copy the key. You won't be able to see it again.": {
    de: "Kopieren Sie den Schlüssel. Sie werden ihn nicht nochmal sehen können."
  },
  "Key name": {
    de: "Name"
  },
  "Close": {
    de: "Schließen"
  },
  "Copy": {
    de: "Kopieren"
  },
  'Are you sure you want to delete this API key "%%"?': {
    de: 'Sind Sie sicher, dass Sie den API-Key "%%" löschen möchten?'
  },
  "API key deleted successfully": {
    de: "API-Key erfolgreich gelöscht"
  },
  "Success": {
    de: "Erfolg"
  },
};

export function translate(s: string) {
  const available = translations[s] ?? {};
  for (const lang of navigator.languages) {
    if (lang === "en" || lang === "en-US") {
      return s;
    }
    const t = available[lang];
    if (t) {
      return t;
    }
  }
  return s;
}
