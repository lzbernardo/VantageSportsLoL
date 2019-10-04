package privileges

const (
	// God is the admin privilege, granted to internal requests, allowing
	// complete read/write access to all APIs. Use carefully.
	VantageGod = "vs_god"

	JstoreRead  = "jstore_read"  // read any (non-user) jstore key
	JstoreWrite = "jstore_write" // write any (non-user) jstore key
	JstoreAdmin = "jstore_admin" // read/write all keys

	MetaRead  = "meta_read"
	MetaWrite = "meta_write"

	LOLstatsRead  = "lolstats_read"
	LOLstatsWrite = "lolstats_write"

	LOLUsersAdmin = "lolusers_admin"

	NBAStatsRead  = "stats_read"
	NBAStatsWrite = "stats_write"

	PaymentAdmin = "payment_admin"

	RiotInternal = "riot_internal"
	RiotExternal = "riot_external"

	TasksEmail      = "tasks_email"
	TasksOcrExtract = "tasks_ocrextract"
	TasksPlaylist   = "tasks_playlist"

	UsersImpersonate = "users_gentoken"
	UsersRead        = "users_read"
	UsersSendToken   = "users_sendtoken"
	UsersWrite       = "users_write"

	VideogenWrite = "videogen_write"
)
