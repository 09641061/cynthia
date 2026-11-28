Feature: Identity, Access Management and Organization Configuration
  As a business owner or administrator
  I want to manage organization settings, team access and API authorization
  So that business operations remain secure and properly configured

  @US05 @US06
  Scenario: Update organization profile and business hours
    Given an authenticated owner belongs to an active organization
    When the owner sends a PATCH request to "/v1/organization" with a valid timezone and schedule
    Then the API responds with status 200
    And the organization profile reflects the updated configuration

  @US27 @US36
  Scenario: Deny unauthenticated access to protected API routes
    Given a client has no active session cookie
    When the client sends a GET request to "/v1/organization"
    Then the API responds with status 401
    And no protected organization data is returned

  @US28 @US29 @US30
  Scenario: Enforce Casbin RBAC permissions for team members
    Given an authenticated member without the "scheduling.appear" permission
    When the system evaluates scheduling visibility for the member
    Then access is controlled according to the organization Casbin policy
