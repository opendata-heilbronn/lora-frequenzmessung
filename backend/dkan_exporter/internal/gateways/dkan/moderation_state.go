package dkan

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

func (c *Client) GetRevisions(datasetID string) error {
	slog.Info("try getting the revisions of dataset", "datasetID", datasetID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	endpoint := fmt.Sprintf("/api/1/metastore/schemas/dataset/items/%s/revisions", datasetID)

	var response any

	slog.Info("do request")

	err := c.do(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		slog.Error("could not get revisions", "err", err)
		return fmt.Errorf("failed to get stuff: %w", err)
	}

	slog.Info("got something", "result", response)

	return nil
}
