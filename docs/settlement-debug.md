# Settlement correction and debugging

Authenticated cashiers can replace their own settlement total with
`PUT /settlement`, body `{"date":"YYYY-MM-DD","actual_cash":33000}`.
The amount is the new total handed in, not an increment. Zero is valid.
Expected cash and discrepancy are recalculated from the cashier's transactions.

Debug reset is enabled for this deployment at the owner's request. Set
`SETTLEMENT_DEBUG_RESET=false` in Render to disable it, or `true` to enable it.
`GET /settlement?date=YYYY-MM-DD` reports `debug_reset_enabled` for the UI.

`DELETE /settlement/debug-reset?date=YYYY-MM-DD` deletes only the authenticated
cashier's settlement for that date. It never deletes transactions or another
cashier's settlement. The endpoint requires authentication and a cashier/admin
role. Repeating a reset is safe; the settlement can then be created again.

For browser automation, use `settlement-actual-cash`, `settlement-save`, and
`settlement-reset` test IDs. Accept the confirmation dialog for reset.
Use dedicated test data; reset permanently removes the targeted settlement.
