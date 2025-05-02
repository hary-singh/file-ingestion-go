CREATE TABLE customer_configs
(
    customer_id        VARCHAR(50) PRIMARY KEY,
    schema_subject     VARCHAR(100) NOT NULL,
    kafka_topic        VARCHAR(100) NOT NULL,
    file_pattern       VARCHAR(100) NOT NULL,
    validation_profile VARCHAR(50)  NOT NULL,
    created_at         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add some test data
INSERT INTO customer_configs (customer_id, schema_subject, kafka_topic, file_pattern, validation_profile)
VALUES ('cust123', 'orders-v1', 'orders.raw', 'orders-*.csv', 'standard_orders'),
       ('cust456', 'claims-v1', 'claims.raw', 'claims-*.csv', 'standard_claims');

-- Add audit trail table for file processing
CREATE TABLE file_processing_audit
(
    id                  SERIAL PRIMARY KEY,
    customer_id         VARCHAR(50)  NOT NULL,
    file_name           VARCHAR(255) NOT NULL,
    records_total       INT          NOT NULL,
    records_valid       INT          NOT NULL,
    status              VARCHAR(50)  NOT NULL,
    error_message       TEXT,
    processing_duration FLOAT        NOT NULL,
    created_at          TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (customer_id) REFERENCES customer_configs (customer_id)
);

-- Add indices
CREATE INDEX idx_audit_customer_id ON file_processing_audit (customer_id);
CREATE INDEX idx_audit_created_at ON file_processing_audit (created_at);