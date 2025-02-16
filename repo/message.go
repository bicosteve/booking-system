package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bicosteve/booking-system/entities"
)

func AddSMSOutbox(ctx context.Context, d *sql.DB, msg entities.SMSPayload) error {
	q := `INSERT INTO sms_outbox(msg, user_id) VALUES(?,?)`

	stmt, err := d.PrepareContext(ctx, q)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return err
	}

	defer stmt.Close()

	text := fmt.Sprintf("Your reset token is '%s' ", msg.Message)

	args := []interface{}{text, msg.UserID}

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return err
	}

	return nil
}
