package application

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/dbaratey/florist-core/internal/inventory/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

type WriteoffBatchCommand struct {
	BatchID kernel.BatchID
	Qty     int
	Reason  string
}

type WriteoffBatchHandler struct {
	repo      BatchRepository
	txRunner  TxRunner
	publisher kernel.EventPublisher
}

func NewWriteoffBatchHandler(repo BatchRepository, tx TxRunner, pub kernel.EventPublisher) *WriteoffBatchHandler {
	return &WriteoffBatchHandler{repo: repo, txRunner: tx, publisher: pub}
}

func (h *WriteoffBatchHandler) Handle(ctx context.Context, cmd WriteoffBatchCommand) error {
	return h.txRunner.RunTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		batch, err := h.repo.FindByID(ctx, cmd.BatchID)
		if err != nil {
			return err
		}

		if err := batch.Writeoff(cmd.Qty, cmd.Reason); err != nil {
			return err
		}

		if err := h.repo.UpdateTx(ctx, tx, batch); err != nil {
			return err
		}
		
		return h.publisher.Publish(ctx, batch.PopEvents()...)
	})
}
