Feature: Service Catalog and Appointment Scheduling
  As an authorized team member
  I want to manage service categories, services and appointments without overlaps
  So that the business maintains an accurate catalog and conflict-free calendar

  @US07 @US08 @US09 @US33
  Scenario: Create a category and service and retrieve the ordered catalog
    Given an authenticated owner belongs to an active organization
    When the owner creates a category via POST "/v1/catalog/categories"
    And creates a service under that category via POST "/v1/catalog/categories/{id}/services"
    And requests GET "/v1/catalog"
    Then the API responds with status 200
    And the response includes the newly created category and service in order

  @US10 @US11 @US12 @US13 @US14
  Scenario: Create, reschedule and cancel an appointment
    Given an active service, customer and available staff member exist
    When an authorized user creates an appointment via POST "/v1/appointments"
    And updates its time slot via PATCH "/v1/appointments/{id}"
    Then the appointment reflects the updated schedule

  @US35
  Scenario: Prevent overlapping active appointments under concurrent requests
    Given a staff member already has a confirmed appointment for a time slot
    When a concurrent POST "/v1/appointments" request targets the same staff member and overlapping time slot
    Then the transactional guard rejects the conflicting booking
    And zero overlapping active appointments exist in PostgreSQL
