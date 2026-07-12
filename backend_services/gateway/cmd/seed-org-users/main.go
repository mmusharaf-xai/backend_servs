package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/eternal-orbit-labs/gateway/internal/config"
	"github.com/eternal-orbit-labs/gateway/internal/platform"
)

func main() {
	godotenv.Load()

	ownerEmail := flag.String("email", "test@gmail.com", "Org owner email")
	appSlug := flag.String("app", "surveillance-pro", "App slug")
	orgID := flag.String("org-id", "", "Organization ID (defaults to owner's most recent org in app)")
	count := flag.Int("count", 150, "Number of member users to seed")
	prefix := flag.String("prefix", "seed-member", "Email local-part prefix for seeded users")
	flag.Parse()

	if *count < 1 {
		log.Fatal("count must be at least 1")
	}

	ctx := context.Background()
	cfg := config.Load()
	pool, err := platform.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	targetOrgID, orgName, err := resolveOrg(ctx, pool, *ownerEmail, *appSlug, *orgID)
	if err != nil {
		log.Fatalf("resolve org: %v", err)
	}

	created, skipped, err := seedMembers(ctx, pool, targetOrgID, *count, *prefix)
	if err != nil {
		log.Fatalf("seed members: %v", err)
	}

	fmt.Fprintf(os.Stdout, "Seeded org %q (%s)\n", orgName, targetOrgID)
	fmt.Fprintf(os.Stdout, "Created %d members, skipped %d existing\n", created, skipped)
}

func resolveOrg(ctx context.Context, pool *pgxpool.Pool, ownerEmail, appSlug, orgID string) (string, string, error) {
	if orgID != "" {
		var name string
		err := pool.QueryRow(ctx, `
			SELECT o.name
			FROM app_organizations o
			JOIN app_org_memberships m ON m.org_id = o.id AND m.role = 'owner'
			JOIN users u ON u.id = m.user_id
			WHERE o.id = $1 AND o.app_slug = $2 AND lower(u.email) = lower($3)
		`, orgID, appSlug, ownerEmail).Scan(&name)
		if err != nil {
			return "", "", fmt.Errorf("org %s not found for %s: %w", orgID, ownerEmail, err)
		}
		return orgID, name, nil
	}

	var id, name string
	err := pool.QueryRow(ctx, `
		SELECT o.id, o.name
		FROM app_organizations o
		JOIN app_org_memberships m ON m.org_id = o.id AND m.role = 'owner'
		JOIN users u ON u.id = m.user_id
		WHERE o.app_slug = $1 AND lower(u.email) = lower($2)
		ORDER BY o.created_at DESC, o.id
		LIMIT 1
	`, appSlug, ownerEmail).Scan(&id, &name)
	if err != nil {
		return "", "", fmt.Errorf("no org owned by %s in app %s: %w", ownerEmail, appSlug, err)
	}
	return id, name, nil
}

func seedMembers(ctx context.Context, pool *pgxpool.Pool, orgID string, count int, prefix string) (created, skipped int, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx)

	for i := 1; i <= count; i++ {
		email := fmt.Sprintf("%s-%03d@seed.eol.dev", prefix, i)
		firstName := "Seed"
		lastName := fmt.Sprintf("User %03d", i)
		phone := fmt.Sprintf("+1555%07d", 1000000+i)

		var userID string
		insertErr := tx.QueryRow(ctx, `
			INSERT INTO users (email, first_name, last_name, phone, email_verified)
			VALUES ($1, $2, $3, $4, true)
			ON CONFLICT (email) DO NOTHING
			RETURNING id
		`, email, firstName, lastName, phone).Scan(&userID)
		if insertErr != nil {
			if errors.Is(insertErr, pgx.ErrNoRows) {
				err = tx.QueryRow(ctx, `SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&userID)
				if err != nil {
					return created, skipped, fmt.Errorf("lookup existing user %s: %w", email, err)
				}
				skipped++
			} else {
				return created, skipped, fmt.Errorf("insert user %s: %w", email, insertErr)
			}
		} else {
			created++
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO app_org_memberships (org_id, user_id, role)
			VALUES ($1, $2, 'member')
			ON CONFLICT (org_id, user_id) DO NOTHING
		`, orgID, userID)
		if err != nil {
			return created, skipped, fmt.Errorf("insert membership for %s: %w", email, err)
		}
		if tag.RowsAffected() == 0 {
			continue
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return created, skipped, err
	}
	return created, skipped, nil
}
