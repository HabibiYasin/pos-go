# Midtrans Sandbox setup

1. Create a merchant account at https://dashboard.midtrans.com/register and switch to Sandbox.
2. Copy the Sandbox API keys from the dashboard's Access Keys page.
3. In Render's environment settings, set `MIDTRANS_SERVER_KEY` and `MIDTRANS_CLIENT_KEY` from the same account, then redeploy.
   The legacy `server_key_mid` name is supported, but the uppercase name takes priority.
   Server keys must never be committed or sent to the browser. The public client key is delivered by `/transaction/payment-config`.
4. In the Midtrans Sandbox dashboard, set Payment Notification URL to:
   `https://pos-go-qcjy.onrender.com/transaction/notification`.
5. Checkout a non-cash order, choose a sandbox payment method, and use the official simulator:
   https://docs.midtrans.com/docs/testing-payment-on-sandbox
6. Verify the notification in the Midtrans dashboard and confirm `paid` in the cashier dashboard.

The application intentionally uses Sandbox only. No real funds are needed.
Missing sandbox keys return a clear unavailable message before creating an order.
The customer retains an order-specific, 48-hour access token in the checkout tab.
Pending payments can be resumed there; a staff login is not required.

Webhooks require a valid SHA-512 signature and an authenticated status lookup to Midtrans.
The verified amount must match the order. Repeated notifications preserve kitchen progress;
card transactions under fraud review are not marked paid.

References:
- https://docs.midtrans.com/docs/https-notification-webhooks
- https://docs.midtrans.com/docs/testing-payment-on-sandbox
