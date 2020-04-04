Feature: QPS limit check
    As a developer am gonna to check if QPS shaper is working as expected

Background: 
    Given local server on port 4010 in debug mode with QPS limit set as 10

Scenario: 
    Given limit of our requests as 5 per sec
    When we send 20 GET requests
    Then we receive 404 code 20 times

Scenario: 
    Given no limit of our requests
    When we send 20 GET requests
    Then we receive 429 code 10 times