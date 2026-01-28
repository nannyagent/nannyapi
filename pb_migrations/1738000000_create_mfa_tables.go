package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ========================================
		// MFA Factors Collection
		// Stores enrolled MFA factors (TOTP) for users
		// ========================================
		mfaFactorsCollection := core.NewBaseCollection("mfa_factors")

		mfaFactorsCollection.Fields.Add(&core.TextField{
			Name:     "user_id",
			Required: true,
		})

		mfaFactorsCollection.Fields.Add(&core.SelectField{
			Name:     "factor_type",
			Required: true,
			Values:   []string{"totp"},
		})

		mfaFactorsCollection.Fields.Add(&core.TextField{
			Name:     "friendly_name",
			Required: false,
			Max:      100,
		})

		mfaFactorsCollection.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"unverified", "verified"},
		})

		// Encrypted TOTP secret (base32 encoded)
		mfaFactorsCollection.Fields.Add(&core.TextField{
			Name:     "secret",
			Required: true,
			Hidden:   true, // Never expose via API
		})

		// Add indexes for efficient lookup
		mfaFactorsCollection.Indexes = []string{
			"CREATE INDEX idx_mfa_factors_user_id ON mfa_factors(user_id)",
			"CREATE INDEX idx_mfa_factors_status ON mfa_factors(status)",
		}

		// API rules - authenticated users can manage their own factors
		mfaFactorsCollection.ListRule = nil   // Only via custom API
		mfaFactorsCollection.ViewRule = nil   // Only via custom API
		mfaFactorsCollection.CreateRule = nil // Only via custom API
		mfaFactorsCollection.UpdateRule = nil // Only via custom API
		mfaFactorsCollection.DeleteRule = nil // Only via custom API

		if err := app.Save(mfaFactorsCollection); err != nil {
			return err
		}

		// ========================================
		// MFA Backup Codes Collection
		// Single-use recovery codes
		// ========================================
		mfaBackupCodesCollection := core.NewBaseCollection("mfa_backup_codes")

		mfaBackupCodesCollection.Fields.Add(&core.TextField{
			Name:     "user_id",
			Required: true,
		})

		// Hashed backup code (bcrypt)
		mfaBackupCodesCollection.Fields.Add(&core.TextField{
			Name:     "code_hash",
			Required: true,
			Hidden:   true,
		})

		mfaBackupCodesCollection.Fields.Add(&core.BoolField{
			Name:     "used",
			Required: false,
		})

		mfaBackupCodesCollection.Fields.Add(&core.DateField{
			Name:     "used_at",
			Required: false,
		})

		// Generation batch ID - all codes generated together share this
		mfaBackupCodesCollection.Fields.Add(&core.TextField{
			Name:     "batch_id",
			Required: true,
		})

		// Add indexes
		mfaBackupCodesCollection.Indexes = []string{
			"CREATE INDEX idx_mfa_backup_codes_user_id ON mfa_backup_codes(user_id)",
			"CREATE INDEX idx_mfa_backup_codes_batch_id ON mfa_backup_codes(batch_id)",
			"CREATE INDEX idx_mfa_backup_codes_used ON mfa_backup_codes(used)",
		}

		// API rules - admin only
		mfaBackupCodesCollection.ListRule = nil
		mfaBackupCodesCollection.ViewRule = nil
		mfaBackupCodesCollection.CreateRule = nil
		mfaBackupCodesCollection.UpdateRule = nil
		mfaBackupCodesCollection.DeleteRule = nil

		if err := app.Save(mfaBackupCodesCollection); err != nil {
			return err
		}

		// ========================================
		// MFA Challenges Collection
		// Tracks verification challenges
		// ========================================
		mfaChallengesCollection := core.NewBaseCollection("mfa_challenges")

		mfaChallengesCollection.Fields.Add(&core.TextField{
			Name:     "factor_id",
			Required: true,
		})

		mfaChallengesCollection.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"pending", "verified", "expired"},
		})

		mfaChallengesCollection.Fields.Add(&core.DateField{
			Name:     "expires_at",
			Required: true,
		})

		mfaChallengesCollection.Fields.Add(&core.DateField{
			Name:     "verified_at",
			Required: false,
		})

		// Add indexes
		mfaChallengesCollection.Indexes = []string{
			"CREATE INDEX idx_mfa_challenges_factor_id ON mfa_challenges(factor_id)",
			"CREATE INDEX idx_mfa_challenges_status ON mfa_challenges(status)",
			"CREATE INDEX idx_mfa_challenges_expires_at ON mfa_challenges(expires_at)",
		}

		// API rules - admin only
		mfaChallengesCollection.ListRule = nil
		mfaChallengesCollection.ViewRule = nil
		mfaChallengesCollection.CreateRule = nil
		mfaChallengesCollection.UpdateRule = nil
		mfaChallengesCollection.DeleteRule = nil

		if err := app.Save(mfaChallengesCollection); err != nil {
			return err
		}

		// ========================================
		// MFA Used Tokens Collection
		// Prevents TOTP token replay attacks
		// ========================================
		mfaUsedTokensCollection := core.NewBaseCollection("mfa_used_tokens")

		mfaUsedTokensCollection.Fields.Add(&core.TextField{
			Name:     "factor_id",
			Required: true,
		})

		// Hash of the used token
		mfaUsedTokensCollection.Fields.Add(&core.TextField{
			Name:     "token_hash",
			Required: true,
		})

		mfaUsedTokensCollection.Fields.Add(&core.DateField{
			Name:     "used_at",
			Required: true,
		})

		// Add indexes
		mfaUsedTokensCollection.Indexes = []string{
			"CREATE INDEX idx_mfa_used_tokens_factor_id ON mfa_used_tokens(factor_id)",
			"CREATE UNIQUE INDEX idx_mfa_used_tokens_unique ON mfa_used_tokens(factor_id, token_hash)",
		}

		// API rules - admin only
		mfaUsedTokensCollection.ListRule = nil
		mfaUsedTokensCollection.ViewRule = nil
		mfaUsedTokensCollection.CreateRule = nil
		mfaUsedTokensCollection.UpdateRule = nil
		mfaUsedTokensCollection.DeleteRule = nil

		if err := app.Save(mfaUsedTokensCollection); err != nil {
			return err
		}

		// ========================================
		// MFA Sensitive Operation Verifications
		// Short-lived verification tokens for sensitive ops
		// ========================================
		mfaSensitiveOpsCollection := core.NewBaseCollection("mfa_sensitive_verifications")

		mfaSensitiveOpsCollection.Fields.Add(&core.TextField{
			Name:     "user_id",
			Required: true,
		})

		mfaSensitiveOpsCollection.Fields.Add(&core.TextField{
			Name:     "operation",
			Required: true,
		})

		mfaSensitiveOpsCollection.Fields.Add(&core.DateField{
			Name:     "valid_until",
			Required: true,
		})

		mfaSensitiveOpsCollection.Fields.Add(&core.BoolField{
			Name:     "used",
			Required: false,
		})

		mfaSensitiveOpsCollection.Fields.Add(&core.DateField{
			Name:     "used_at",
			Required: false,
		})

		// Add indexes
		mfaSensitiveOpsCollection.Indexes = []string{
			"CREATE INDEX idx_mfa_sensitive_verifications_user_id ON mfa_sensitive_verifications(user_id)",
			"CREATE INDEX idx_mfa_sensitive_verifications_valid_until ON mfa_sensitive_verifications(valid_until)",
		}

		// API rules - admin only
		mfaSensitiveOpsCollection.ListRule = nil
		mfaSensitiveOpsCollection.ViewRule = nil
		mfaSensitiveOpsCollection.CreateRule = nil
		mfaSensitiveOpsCollection.UpdateRule = nil
		mfaSensitiveOpsCollection.DeleteRule = nil

		if err := app.Save(mfaSensitiveOpsCollection); err != nil {
			return err
		}

		// ========================================
		// Add mfa_enabled field to users collection
		// ========================================
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Check if field already exists
		if usersCollection.Fields.GetByName("mfa_enabled") == nil {
			usersCollection.Fields.Add(&core.BoolField{
				Name:     "mfa_enabled",
				Required: false,
			})

			if err := app.Save(usersCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback
		collections := []string{
			"mfa_sensitive_verifications",
			"mfa_used_tokens",
			"mfa_challenges",
			"mfa_backup_codes",
			"mfa_factors",
		}

		for _, name := range collections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err == nil {
				if delErr := app.Delete(collection); delErr != nil {
					return delErr
				}
			}
		}

		// Remove mfa_enabled field from users
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err == nil {
			if field := usersCollection.Fields.GetByName("mfa_enabled"); field != nil {
				usersCollection.Fields.RemoveByName("mfa_enabled")
				app.Save(usersCollection)
			}
		}

		return nil
	})
}
