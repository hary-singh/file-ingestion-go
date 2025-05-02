package adapters

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"validator-function/internal/domain"
)

type PostgresConfigRepository struct {
	db *sql.DB
}

func NewPostgresConfigRepository(connectionString string) (*PostgresConfigRepository, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresConfigRepository{
		db: db,
	}, nil
}

func (r *PostgresConfigRepository) GetCustomerConfig(ctx context.Context, customerID string) (*domain.CustomerConfig, error) {
	query := `
        SELECT customer_id, schema_subject, kafka_topic, file_pattern, validation_profile
        FROM customer_configs
        WHERE customer_id = $1
    `

	row := r.db.QueryRowContext(ctx, query, customerID)

	config := &domain.CustomerConfig{}
	err := row.Scan(
		&config.CustomerID,
		&config.SchemaSubject,
		&config.KafkaTopic,
		&config.FilePattern,
		&config.ValidationProfile,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no config found for customer %s", customerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan config: %w", err)
	}

	return config, nil
}
