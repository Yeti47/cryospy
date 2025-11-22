module cryospy-e2e

go 1.24.3

require (
	github.com/mattn/go-sqlite3 v1.14.24
	github.com/yeti47/cryospy/server/core v0.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
)

replace github.com/yeti47/cryospy/server/core => ../../server/core
