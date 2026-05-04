package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/repository"
)

type SyncMetaAccount struct {
	meta      metaads.Client
	bmRepo    repository.BusinessManagerRepository
	accRepo   repository.MetaAdAccountRepository
	pageRepo  repository.MetaPageRepository
	pixelRepo repository.MetaPixelRepository
	tokens    repository.MetaTokenRepository
	creds     repository.AppCredentialRepository
	cfg       *config.Service
}

func NewSyncMetaAccount(
	meta metaads.Client,
	bmRepo repository.BusinessManagerRepository,
	accRepo repository.MetaAdAccountRepository,
	pageRepo repository.MetaPageRepository,
	pixelRepo repository.MetaPixelRepository,
	tokens repository.MetaTokenRepository,
	creds repository.AppCredentialRepository,
	cfg *config.Service,
) *SyncMetaAccount {
	return &SyncMetaAccount{
		meta: meta, bmRepo: bmRepo, accRepo: accRepo,
		pageRepo: pageRepo, pixelRepo: pixelRepo,
		tokens: tokens, creds: creds, cfg: cfg,
	}
}

// Run pulls the entire Meta hierarchy for the user's active token and persists
// it. Top-level errors (no token, businesses fetch failed) abort the sync.
// Per-BM failures are logged and skipped.
func (u *SyncMetaAccount) Run(ctx context.Context, userID string) error {
	token, err := u.tokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("active meta token: %w", err)
	}
	access := token.PlainToken
	if access == "" {
		return fmt.Errorf("meta token: empty plaintext")
	}

	// Recover app secret (may be empty if user did not provide one — appsecret_proof skipped).
	var appSecret string
	if token.AppID != "" {
		cred, err := u.creds.GetByUserAndAppID(ctx, userID, token.AppID)
		if err == nil {
			if plain, decErr := u.cfg.Decrypt(cred.EncryptedAppSecret); decErr == nil {
				appSecret = plain
			}
		}
	}

	// Top-level: businesses + personal accounts in parallel.
	var (
		businesses []metaads.Business
		personal   []metaads.AdAccountFull
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		bms, err := u.meta.GetBusinesses(gctx, access, appSecret)
		if err != nil {
			return fmt.Errorf("list businesses: %w", err)
		}
		businesses = bms
		return nil
	})
	g.Go(func() error {
		accs, err := u.meta.GetMyAdAccounts(gctx, access, appSecret)
		if err != nil {
			slog.Warn("meta sync: my adaccounts failed", "user_id", userID, "err", err)
			return nil
		}
		personal = accs
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}

	// Persist BMs first so we can FK ad accounts to them.
	bmDBIDs := make(map[string]string, len(businesses))
	for _, b := range businesses {
		raw, _ := json.Marshal(b)
		row := &domain.BusinessManager{
			MetaID:             b.ID,
			UserID:             userID,
			Name:               b.Name,
			VerificationStatus: b.VerificationStatus,
			TimezoneID:         b.TimezoneID,
			Vertical:           b.Vertical,
			Raw:                raw,
		}
		if err := u.bmRepo.Upsert(ctx, row); err != nil {
			slog.Warn("meta sync: bm upsert failed", "bm", b.ID, "err", err)
			continue
		}
		bmDBIDs[b.ID] = row.ID
	}

	// Persist personal ad accounts (no BM).
	for _, a := range personal {
		acc := metaAccountToDomain(a, nil, userID, domain.AccessKindPersonal)
		if err := u.accRepo.Upsert(ctx, acc); err != nil {
			slog.Warn("meta sync: personal account upsert failed", "act", a.ID, "err", err)
		}
	}

	// Per-BM fan-out (sem of 6).
	sem := make(chan struct{}, 6)
	bmGroup, bmCtx := errgroup.WithContext(ctx)
	for _, bm := range businesses {
		bm := bm
		bmGroup.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			bmDBID, ok := bmDBIDs[bm.ID]
			if !ok {
				return nil
			}
			u.syncBM(bmCtx, userID, access, appSecret, bm.ID, bmDBID)
			return nil
		})
	}
	if err := bmGroup.Wait(); err != nil {
		slog.Warn("meta sync: per-bm group ended with error", "err", err)
	}
	return nil
}

// syncBM fetches accounts/pages/pixels/IG for one BM in parallel. Errors are
// logged and tolerated.
func (u *SyncMetaAccount) syncBM(ctx context.Context, userID, access, appSecret, bmMetaID, bmDBID string) {
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		accs, err := u.meta.GetBMOwnedAccounts(gctx, access, appSecret, bmMetaID)
		if err != nil {
			slog.Warn("meta sync: owned_ad_accounts failed", "bm", bmMetaID, "err", err)
			return nil
		}
		for _, a := range accs {
			acc := metaAccountToDomain(a, &bmDBID, userID, domain.AccessKindOwned)
			if err := u.accRepo.Upsert(gctx, acc); err != nil {
				slog.Warn("meta sync: account upsert failed", "act", a.ID, "err", err)
			}
		}
		return nil
	})
	g.Go(func() error {
		accs, err := u.meta.GetBMClientAccounts(gctx, access, appSecret, bmMetaID)
		if err != nil {
			slog.Warn("meta sync: client_ad_accounts failed", "bm", bmMetaID, "err", err)
			return nil
		}
		for _, a := range accs {
			acc := metaAccountToDomain(a, &bmDBID, userID, domain.AccessKindClient)
			if err := u.accRepo.Upsert(gctx, acc); err != nil {
				slog.Warn("meta sync: account upsert failed", "act", a.ID, "err", err)
			}
		}
		return nil
	})
	g.Go(func() error {
		pages, err := u.meta.GetBMPages(gctx, access, appSecret, bmMetaID)
		if err != nil {
			slog.Warn("meta sync: pages failed", "bm", bmMetaID, "err", err)
			return nil
		}
		for _, p := range pages {
			raw, _ := json.Marshal(p)
			row := &domain.MetaPage{
				MetaID:             p.ID,
				BMID:               &bmDBID,
				UserID:             userID,
				Name:               p.Name,
				Category:           p.Category,
				FanCount:           p.FanCount,
				EncryptedPageToken: p.AccessToken, // gets encrypted by repo
				Raw:                raw,
			}
			if err := u.pageRepo.Upsert(gctx, row); err != nil {
				slog.Warn("meta sync: page upsert failed", "page", p.ID, "err", err)
			}
		}
		return nil
	})
	g.Go(func() error {
		pixels, err := u.meta.GetBMPixels(gctx, access, appSecret, bmMetaID)
		if err != nil {
			slog.Warn("meta sync: pixels failed", "bm", bmMetaID, "err", err)
			return nil
		}
		for _, p := range pixels {
			raw, _ := json.Marshal(p)
			row := &domain.MetaPixel{
				MetaID:   p.ID,
				BMID:     &bmDBID,
				UserID:   userID,
				Name:     p.Name,
				IsActive: !p.IsUnavailable,
				Raw:      raw,
			}
			if t := parseMetaTime(p.LastFiredTime); !t.IsZero() {
				row.LastFired = &t
			}
			if err := u.pixelRepo.Upsert(gctx, row); err != nil {
				slog.Warn("meta sync: pixel upsert failed", "pixel", p.ID, "err", err)
			}
		}
		return nil
	})
	g.Go(func() error {
		// Instagram persistence is logged but not yet stored — table exists,
		// repo for it is intentionally minimal in F1.
		_, err := u.meta.GetBMInstagram(gctx, access, appSecret, bmMetaID)
		if err != nil {
			slog.Warn("meta sync: instagram failed", "bm", bmMetaID, "err", err)
		}
		return nil
	})
	_ = g.Wait()
}

// metaAccountToDomain converts the Graph API shape into our DB row.
func metaAccountToDomain(a metaads.AdAccountFull, bmID *string, userID, accessKind string) *domain.MetaAdAccount {
	raw, _ := json.Marshal(a)
	return &domain.MetaAdAccount{
		MetaID:        a.ID,
		BMID:          bmID,
		UserID:        userID,
		Name:          a.Name,
		Currency:      a.Currency,
		TimezoneName:  a.TimezoneName,
		AccountStatus: a.AccountStatus,
		DisableReason: a.DisableReason,
		SpendCap:      parseMoney(a.SpendCap),
		AmountSpent:   parseMoney(a.AmountSpent),
		Balance:       parseMoney(a.Balance),
		AccessKind:    accessKind,
		Raw:           raw,
	}
}

func parseMoney(s string) float64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseMetaTime parses Graph API timestamps (ISO 8601 with offset).
func parseMetaTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-0700"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
