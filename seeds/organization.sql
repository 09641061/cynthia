UPDATE organizations
SET
    whatsapp_phone_number = '+51994356103',
    whatsapp_phone_number_id = '1414143045107488',
    whatsapp_enabled = TRUE,
    ai_enabled = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE singleton_key = TRUE;
