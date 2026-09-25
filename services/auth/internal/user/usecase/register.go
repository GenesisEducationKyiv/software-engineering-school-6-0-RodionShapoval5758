package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/auth/internal/db"
	"GithubReleaseNotificationAPI/services/auth/internal/user"

	"github.com/jackc/pgx/v5"
)

const minPasswordLen = 8

type registerStore interface {
	CreateUser(ctx context.Context, q db.DBTX, email, passwordHash string) (string, error)
	InsertVerification(ctx context.Context, q db.DBTX, token, userID string, expiresAt time.Time) error
}

type outboxInserter interface {
	Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error
}

type Register struct {
	store     registerStore
	outbox    outboxInserter
	txer      db.TxBeginner
	verifyTTL time.Duration
}

func NewRegister(store registerStore, outbox outboxInserter, txer db.TxBeginner, verifyTTL time.Duration) *Register {
	return &Register{store: store, outbox: outbox, txer: txer, verifyTTL: verifyTTL}
}

func (r *Register) Execute(ctx context.Context, email, password string) error {
	email, err := normalizeEmail(email)
	if err != nil {
		return err
	}

	if len(password) < minPasswordLen {
		return user.ErrWeakPassword
	}

	passwordHash, err := user.HashPassword(password)
	if err != nil {
		return err
	}

	verifyToken, err := newRawToken()
	if err != nil {
		return err
	}

	payload, err := json.Marshal(contract.VerificationRequested{
		Email:       email,
		VerifyToken: verifyToken,
	})
	if err != nil {
		return fmt.Errorf("marshal verification event: %w", err)
	}

	tx, err := r.txer.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin register tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	userID, err := r.store.CreateUser(ctx, tx, email, passwordHash)
	if err != nil {
		return err
	}

	if err := r.store.InsertVerification(ctx, tx, verifyToken, userID, time.Now().Add(r.verifyTTL)); err != nil {
		return err
	}

	if err := r.outbox.Insert(ctx, tx, contract.SubjectVerifyEmail, payload); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", user.ErrInvalidEmailFormat
	}

	return email, nil
}
