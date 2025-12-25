package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// PocketBase superusers (admins) cannot be created programmatically in migrations
		// To create admin, run: ./nannyapi superuser upsert admin@nannyapi.local AdminPass-123
		return nil
	}, nil)
}
