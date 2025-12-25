package pb_migrations

import (
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Get or create users collection
		collection, _ := app.FindCollectionByNameOrId("users")
		isNew := collection == nil

		if isNew {
			// Create auth collection with OAuth2 providers
			collection = core.NewAuthCollection("users")

			// Enable password authentication
			collection.PasswordAuth.Enabled = true
			collection.PasswordAuth.IdentityFields = []string{"email"}
		}

		// Configure OAuth2 (always update in case credentials changed)
		githubClientId := os.Getenv("GITHUB_CLIENT_ID")
		githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
		googleClientId := os.Getenv("GOOGLE_CLIENT_ID")
		googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

		// Only enable OAuth2 if credentials are provided
		if (githubClientId != "" && githubClientSecret != "") || (googleClientId != "" && googleClientSecret != "") {
			collection.OAuth2.Enabled = true

			var providers []core.OAuth2ProviderConfig
			if githubClientId != "" && githubClientSecret != "" {
				providers = append(providers, core.OAuth2ProviderConfig{
					Name:         "github",
					ClientId:     githubClientId,
					ClientSecret: githubClientSecret,
				})
			}
			if googleClientId != "" && googleClientSecret != "" {
				providers = append(providers, core.OAuth2ProviderConfig{
					Name:         "google",
					ClientId:     googleClientId,
					ClientSecret: googleClientSecret,
				})
			}
			collection.OAuth2.Providers = providers

			// Map OAuth2 fields
			collection.OAuth2.MappedFields.Name = "name"
			collection.OAuth2.MappedFields.Username = "username"
			collection.OAuth2.MappedFields.AvatarURL = "avatar"
		}

		// Add custom fields only if new
		if isNew {
			collection.Fields.Add(&core.TextField{
				Name:     "name",
				Required: false,
				Max:      255,
			})

			// Track OAuth signup
			collection.Fields.Add(&core.BoolField{
				Name:     "oauth_signup",
				Required: false,
			})

			collection.Fields.Add(&core.FileField{
				Name:      "avatar",
				MaxSelect: 1,
				MaxSize:   5242880, // 5MB
				MimeTypes: []string{
					"image/jpeg",
					"image/png",
					"image/svg+xml",
					"image/gif",
					"image/webp",
				},
			})

			// Set API rules
			collection.ListRule = types.Pointer("id = @request.auth.id")
			collection.ViewRule = types.Pointer("id = @request.auth.id")
			collection.CreateRule = types.Pointer("") // Anyone can sign up
			collection.UpdateRule = types.Pointer("id = @request.auth.id")
			collection.DeleteRule = types.Pointer("id = @request.auth.id")
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: delete the users collection
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
