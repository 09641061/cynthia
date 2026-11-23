# WhatsApp channel

On an incoming message, validate the provider notification, normalize its phone, find the customer or create one using available data, create or recover the appropriate conversation, and store the message without duplicates. Reopen a closed conversation. An existing manually registered customer with a matching number is reused.

A user replies from an available conversation. Request delivery through WhatsApp and record the result, retaining failed messages. Provider restrictions and uncertain outcomes must be handled explicitly; never assume a retry is safe without reconciliation or a supported idempotency mechanism.

When WhatsApp is unavailable, customers can still be registered, searched and edited, and stored conversations can be consulted. Sending and receiving depend on the channel recovering. Do not make WhatsApp the source of truth for customer identity or stored history.
