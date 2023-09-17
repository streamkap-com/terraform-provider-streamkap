package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Source struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	SubId        string `json:"sub_id"`
	TenantId     string `json:"tenant_id"`
	Connector    string `json:"connector"`
	TaskStatuses struct {
		Field1 struct {
			Status string `json:"status"`
		} `json:"0"`
	} `json:"task_statuses"`
	Tasks           []int    `json:"tasks"`
	ConnectorStatus string   `json:"connector_status"`
	TopicIds        []string `json:"topic_ids"`
	Topics          []string `json:"topics"`
	InlineMetrics   struct {
		ConnectorStatus []struct {
			Timestamp string `json:"timestamp"`
			Value     string `json:"value"`
		} `json:"connector_status"`
		Latency []struct {
			Timestamp string `json:"timestamp"`
			Value     string `json:"value"`
		} `json:"latency"`
		SnapshotState []struct {
			Timestamp string `json:"timestamp"`
			Value     string `json:"value"`
		} `json:"snapshotState"`
		State []struct {
			Timestamp string `json:"timestamp"`
			Value     string `json:"value"`
		} `json:"state"`
		StreamingState []struct {
			Timestamp string `json:"timestamp"`
			Value     string `json:"value"`
		} `json:"streamingState"`
		SourceRecordWriteTotal []struct {
			Timestamp string `json:"timestamp"`
			Value     int    `json:"value"`
		} `json:"sourceRecordWriteTotal"`
	} `json:"inline_metrics"`
	Server string `json:"server"`
	Config struct {
		Key string `json:"key"`
	} `json:"config"`
}

type CreateSourceConfig struct {
	DatabaseHostnameUserDefined          string `json:"database.hostname.user.defined"`
	DatabasePort                         string `json:"database.port"`
	DatabaseUser                         string `json:"database.user"`
	DatabasePassword                     string `json:"database.password"`
	DatabaseIncludeListUserDefined       string `json:"database.include.list.user.defined"`
	TableIncludeListUserDefined          string `json:"table.include.list.user.defined"`
	SignalDataCollectionSchemaOrDatabase string `json:"signal.data.collection.schema.or.database"`
	DatabaseConnectionTimeZone           string `json:"database.connectionTimeZone"`
	SnapshotGtid                         string `json:"snapshot.gtid"`
	SnapshotModeUserDefined              string `json:"snapshot.mode.user.defined"`
	BinaryHandlingMode                   string `json:"binary.handling.mode"`
	IncrementalSnapshotChunkSize         int    `json:"incremental.snapshot.chunk.size"`
	MaxBatchSize                         int    `json:"max.batch.size"`
}

type CreateSourceRequest struct {
	Name      string             `json:"name"`
	Connector string             `json:"connector"`
	Config    CreateSourceConfig `json:"config"`
}

type CreateSourceResponse struct {
	Cursor struct {
		TimestampFrom time.Time `json:"timestamp_from"`
		TimestampTo   time.Time `json:"timestamp_to"`
		TopicListFrom int       `json:"topic_list_from"`
		TopicListSize int       `json:"topic_list_size"`
		TopicListSort string    `json:"topic_list_sort"`
	} `json:"cursor"`
	EntityType string   `json:"entity_type"`
	Data       []Source `json:"data"`
}

type SourceConfigurationResponse struct {
	Connector             string   `json:"connector"`
	DisplayName           string   `json:"display_name"`
	Status                string   `json:"status"`
	SchemaLevels          []string `json:"schema_levels"`
	DebeziumConnectorName string   `json:"debezium_connector_name"`
	Serialisation         string   `json:"serialisation"`
	Metrics               []struct {
		Attribute   string `json:"attribute"`
		Aggregation string `json:"aggregation"`
		Value       struct {
			Dynamic      bool     `json:"dynamic"`
			FunctionName string   `json:"function_name"`
			Dependencies []string `json:"dependencies,omitempty"`
		} `json:"value,omitempty"`
		Display struct {
			Name  string `json:"name"`
			Unit  string `json:"unit"`
			Pages struct {
				Sources struct {
					DisplayByDefault bool     `json:"display_by_default"`
					Levels           []string `json:"levels"`
				} `json:"sources"`
				Pipelines struct {
					DisplayByDefault bool     `json:"display_by_default"`
					Levels           []string `json:"levels"`
				} `json:"pipelines,omitempty"`
				Subscriptions struct {
					DisplayByDefault bool     `json:"display_by_default"`
					Levels           []string `json:"levels"`
				} `json:"subscriptions,omitempty"`
			} `json:"pages"`
		} `json:"display"`
		Mbean                string `json:"mbean,omitempty"`
		ClickhouseMetricName string `json:"clickhouse_metric_name,omitempty"`
		Level                string `json:"level,omitempty"`
		LabelAsValue         string `json:"label_as_value,omitempty"`
		Context              string `json:"context,omitempty"`
		Category             string `json:"category,omitempty"`
	} `json:"metrics"`
	Config []struct {
		Name        string `json:"name"`
		UserDefined bool   `json:"user_defined"`
		Value       struct {
			Type         string      `json:"type,omitempty"`
			RawValue     interface{} `json:"raw_value,omitempty"`
			Control      string      `json:"control,omitempty"`
			FunctionName string      `json:"function_name,omitempty"`
			RawValues    []string    `json:"raw_values,omitempty"`
			Default      interface{} `json:"default,omitempty"`
			Dependencies []string    `json:"dependencies,omitempty"`
			Max          int         `json:"max,omitempty"`
			Min          int         `json:"min,omitempty"`
			Step         int         `json:"step,omitempty"`
		} `json:"value"`
		KafkaConfig      bool   `json:"kafka_config"`
		Description      string `json:"description,omitempty"`
		OrderOfDisplay   int    `json:"order_of_display,omitempty"`
		Required         bool   `json:"required,omitempty"`
		DisplayName      string `json:"display_name,omitempty"`
		Tab              string `json:"tab,omitempty"`
		Encrypt          bool   `json:"encrypt,omitempty"`
		SchemaLevel      string `json:"schema_level,omitempty"`
		SchemaNameFormat string `json:"schema_name_format,omitempty"`
	} `json:"config"`
}

func (s *streamkapAPI) CreateSource(ctx context.Context, reqPayload CreateSourceRequest) (*CreateSourceResponse, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/api/sources", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp CreateSourceResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) ListSourceConfigurations(ctx context.Context) ([]SourceConfigurationResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/api/list-sources", http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp []SourceConfigurationResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
