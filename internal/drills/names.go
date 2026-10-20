package drills

// Name lists that drills fill their placeholders from. Lists used together
// in one drill must not share a name; a test checks this.
var (
	topDirs     = []string{"projects", "code", "repos", "dev"}
	subDirs     = []string{"web", "site", "app", "blog"}
	sideDirs    = []string{"api", "server", "backend", "cli"}
	srcDirs     = []string{"src", "lib", "assets", "styles"}
	mediaDirs   = []string{"music", "photos", "books", "games"}
	keepDirs    = []string{"old", "new", "best", "favourites"}
	jobDirs     = []string{"work", "clients", "school", "admin"}
	years       = []string{"2022", "2023", "2024", "2025"}
	months      = []string{"january", "march", "june", "october"}
	quarters    = []string{"q1", "q2", "q3", "q4"}
	trips       = []string{"beach", "city", "family", "hiking"}
	apps        = []string{"keydrill", "tracker", "budget", "website"}
	spaced      = []string{"my files", "old stuff", "to sort", "client work"}
	spaced2     = []string{"summer 2024", "first draft", "tax return", "road trip"}
	hiddenDirs  = []string{".vault", ".secret", ".stash", ".private"}
	configDirs  = []string{".config", ".local", ".prefs", ".data"}
	hiddenFiles = []string{".env", ".history", ".profile", ".settings"}
	fileNames   = []string{"report", "budget", "invoice", "minutes"}
)
