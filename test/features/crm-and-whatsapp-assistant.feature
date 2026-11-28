Feature: CRM, WhatsApp Webhooks and AI Assistant Coordination
  As a business team member or end customer
  I want to manage customers, conversations and WhatsApp automated assistance reliably
  So that inquiries and bookings are handled without duplicate messages

  @US20 @US21 @US22 @US34
  Scenario: Create and search a customer via CRM API
    Given an authenticated member belongs to an active organization
    When the member creates a customer via POST "/v1/customers"
    And searches by phone number via GET "/v1/customers"
    Then the API returns the matching customer record

  @US23 @US24 @US25 @US26
  Scenario: Manage conversation status and pause AI assistant replies
    Given an active customer conversation exists in CRM
    When an authorized member sends PATCH "/v1/conversations/{id}/ai" with ai_paused set to true
    Then automatic AI replies are paused for that conversation
    And the member can reply directly via POST "/v1/conversations/{id}/messages"

  @US15 @US16 @US17 @US18 @US19 @US31 @US32
  Scenario: Receive an inbound WhatsApp webhook event idempotently
    Given WhatsApp Cloud API delivers an inbound customer message to POST "/v1/crm/webhooks/whatsapp"
    When the same external message identifier is delivered a second time
    Then CRM preserves a single logical message record
    And the AI Assistant worker processes the request without duplicating outbound messages
