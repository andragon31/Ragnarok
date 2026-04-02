package unified

const (
	// Error messages
	errFailedParseParams = "failed to parse params: %w"
	errToolNotFound     = "tool not found: %s"
	errNotFound         = "not found: %s"
	
	// DB Names
	dbFenrir = "fenrir.db"
	dbHati   = "hati.db"
	dbSkoll  = "skoll.db"
	dbTyr    = "tyr.db"
	
	// Step prefixes
	stepPhaseCreate = "phase_create:"
	stepTaskStart   = "task_start:"
	
	// Log messages
	logParsePRD     = "Fenrir: Parse PRD Requirements"
	logGeneratePlan = "Hati: Generate Development Plan"
	logExecutionPlan = " Execution Plan"
	
	// Status
	statusSuccess   = "success"
	statusCompleted = "completed"
)
