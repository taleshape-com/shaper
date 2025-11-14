// SPDX-License-Identifier: MPL-2.0

// Currently supporting English and German.
// Only translating customer-facing features.
// Happy to add new languages if anyone needs something.
const translations = {
  "No Data": {
    de: "Keine Daten",
  },
  "Nothing to show yet": {
    de: "Noch nichts zu zeigen",
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
  "Total": {
    de: "Summe",
  },
  "Save as image": {
    de: "Als Bild speichern",
  },
  "in %% seconds": {
    de: "in %% Sekunden",
  },
  "in %% minutes": {
    de: "in %% Minuten",
  },
  "in %% hours": {
    de: "in %% Stunden",
  },
  "in %% days": {
    de: "in %% Tagen",
  },
  "%% seconds ago": {
    de: "vor %% Sekunden",
  },
  "%% minutes ago": {
    de: "vor %% Minuten",
  },
  "%% hours ago": {
    de: "vor %% Stunden",
  },
  "%% days ago": {
    de: "vor %% Tagen",
  },
  "Password Required": {
    de: "Passwort erforderlich",
  },
  "This dashboard is password protected. Please enter the password to continue.": {
    de: "Dieses Dashboard ist passwortgeschützt. Bitte geben Sie das Passwort ein, um fortzufahren.",
  },
  "Enter password": {
    de: "Passwort eingeben",
  },
  "Access Dashboard": {
    de: "Dashboard öffnen",
  },
  "Invalid password. Please try again.": {
    de: "Ungültiges Passwort. Bitte versuchen Sie es erneut.",
  },
  "Dashboard is not public": {
    de: "Dashboard ist nicht öffentlich",
  },
  "Failed to get JWT for password-protected dashboard": {
    de: "Fehler beim Laden des JWTs",
  },
  "Failed to retrieve JWT for public dashboard": {
    de: "Fehler beim Laden des JWTs",
  },
  "Select all": {
    de: "Alle auswählen",
  },
  "Unselect all": {
    de: "Alle abwählen",
  },
  "Minimum": {
    de: "Minimum",
  },
  "Q1": {
    de: "Q1",
  },
  "Median": {
    de: "Median",
  },
  "Q3": {
    de: "Q3",
  },
  "Maximum": {
    de: "Maximum",
  },
};

export function translate(s: keyof typeof translations) {
  const available = translations[s] ?? {};
  for (const lang of navigator.languages) {
    const firstPart = lang.split("-")[0];
    if (firstPart === "en") {
      return s;
    }
    // TODO: The type casting is more hack than accurate.
    const t = available[firstPart as "de"];
    if (t) {
      return t;
    }
  }
  return s;
}
