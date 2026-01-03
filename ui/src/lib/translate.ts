// SPDX-License-Identifier: MPL-2.0

// Currently supporting English, German, and Portuguese (Brazilian).
// Only translating customer-facing features.
// Happy to add new languages if anyone needs something.
const translations = {
  "No Data": {
    de: "Keine Daten",
    pt: "Sem Dados",
  },
  "Nothing to show yet": {
    de: "Noch nichts zu zeigen",
    pt: "Nada para mostrar ainda",
  },
  Apply: {
    de: "Bestätigen",
    pt: "Aplicar",
  },
  Cancel: {
    de: "Abbrechen",
    pt: "Cancelar",
  },
  "Select date": {
    de: "Datum wählen",
    pt: "Selecionar data",
  },
  "Select date range": {
    de: "Zeitraum wählen",
    pt: "Selecionar período",
  },
  Today: {
    de: "Heute",
    pt: "Hoje",
  },
  "Last 7 days": {
    de: "Letzte 7 Tage",
    pt: "Últimos 7 dias",
  },
  "Last 30 days": {
    de: "Letzte 30 Tage",
    pt: "Últimos 30 dias",
  },
  "Last 3 months": {
    de: "Letzte 3 Monate",
    pt: "Últimos 3 meses",
  },
  "Last 6 months": {
    de: "Letzte 6 Monate",
    pt: "Últimos 6 meses",
  },
  "Month to date": {
    de: "Monat bis heute",
    pt: "Mês até hoje",
  },
  "Year to date": {
    de: "Jahr bis heute",
    pt: "Ano até hoje",
  },
  "Total": {
    de: "Summe",
    pt: "Total",
  },
  "Save as image": {
    de: "Als Bild speichern",
    pt: "Salvar como imagem",
  },
  "in %% seconds": {
    de: "in %% Sekunden",
    pt: "em %% segundos",
  },
  "in %% minutes": {
    de: "in %% Minuten",
    pt: "em %% minutos",
  },
  "in %% hours": {
    de: "in %% Stunden",
    pt: "em %% horas",
  },
  "in %% days": {
    de: "in %% Tagen",
    pt: "em %% dias",
  },
  "%% seconds ago": {
    de: "vor %% Sekunden",
    pt: "há %% segundos",
  },
  "%% minutes ago": {
    de: "vor %% Minuten",
    pt: "há %% minutos",
  },
  "%% hours ago": {
    de: "vor %% Stunden",
    pt: "há %% horas",
  },
  "%% days ago": {
    de: "vor %% Tagen",
    pt: "há %% dias",
  },
  "Password Required": {
    de: "Passwort erforderlich",
    pt: "Senha Necessária",
  },
  "This dashboard is password protected. Please enter the password to continue.": {
    de: "Dieses Dashboard ist passwortgeschützt. Bitte geben Sie das Passwort ein, um fortzufahren.",
    pt: "Este painel está protegido por senha. Por favor, insira a senha para continuar.",
  },
  "Enter password": {
    de: "Passwort eingeben",
    pt: "Inserir senha",
  },
  "Access Dashboard": {
    de: "Dashboard öffnen",
    pt: "Acessar Painel",
  },
  "Invalid password. Please try again.": {
    de: "Ungültiges Passwort. Bitte versuchen Sie es erneut.",
    pt: "Senha inválida. Por favor, tente novamente.",
  },
  "Dashboard is not public": {
    de: "Dashboard ist nicht öffentlich",
    pt: "Painel não é público",
  },
  "Failed to get JWT for password-protected dashboard": {
    de: "Fehler beim Laden des JWTs",
    pt: "Falha ao obter JWT para painel protegido por senha",
  },
  "Failed to retrieve JWT for public dashboard": {
    de: "Fehler beim Laden des JWTs",
    pt: "Falha ao recuperar JWT para painel público",
  },
  "Select all": {
    de: "Alle auswählen",
    pt: "Selecionar todos",
  },
  "Unselect all": {
    de: "Alle abwählen",
    pt: "Desselecionar todos",
  },
  "min": {
    de: "Min.",
    pt: "mín",
  },
  "Q1": {
    de: "Q1",
    pt: "Q1",
  },
  "median": {
    de: "Median",
    pt: "mediana",
  },
  "Q3": {
    de: "Q3",
    pt: "Q3",
  },
  "max": {
    de: "Max.",
    pt: "máx",
  },
  "Other": {
    de: "Andere",
    pt: "Outros",
  },
  "Breakdown": {
    de: "Aufschlüsselung",
    pt: "Detalhamento",
  },
};

export function translate (s: keyof typeof translations) {
  const available = translations[s] ?? {};
  for (const lang of navigator.languages) {
    const firstPart = lang.split("-")[0];
    if (firstPart === "en") {
      return s;
    }
    // TODO: The type casting is more hack than accurate.
    const t = available[firstPart as "de" | "pt"];
    if (t) {
      return t;
    }
  }
  return s;
}
