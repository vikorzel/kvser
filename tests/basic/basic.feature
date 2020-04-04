Feature: basic functionality
  As a developer am gona to check 
  If basic functionality of the server is working properly

Background: 
  Given local server on port 4000 in debug mode

Scenario: Set first time
  When we send POST with key as "case1_<key>" and valid JSON as body
  Then the response code will be 201
  And the response body will be contain "case1_<key>"
Examples:
  | key |
  | 123 |
  | abc |
  | ___ |


Scenario: Set twice
  When we send POST with key as "case2_<key>" and valid JSON as body
  And we send POST with key as "case2_<key>" and valid JSON as body
  Then the response code will be 409
  And the response body will be contain "case2_<key>"
Examples:
  | key |
  | 123 |
  | abc |
  | ___ |

Scenario: Set wrong data
  When we send POST with key as "case3" and invalid JSON as body
  Then the response code will be 400
  And the response body will be contain "Cannot parse JSON"

Scenario: Get
 When we send POST with key as "case4_<post_key>" and valid JSON as body
 And we send GET with key as "case4_<get_key>"
 Then the response code will be <code>
Examples:
 | post_key | get_key   | code |
 | correct1 | correct1  | 200  |
 | correct2 | incorrect | 404  |

Scenario: Get return same data
 When we send POST with key as "case5" and uniq JSON as body
 And we send GET with key as "case5"
 Then the response code will be 200
 And the response body will contains the same JSON

Scenario: Put
 When we send POST with key as "case6_<post_key>" and valid JSON as body
 And we send PUT with key as "case6_<put_key>" and <json_valid_type> JSON as body
 Then the response code will be <code>
 And the response body will be contain "<resp_body>"
Examples:
  | post_key | put_key | json_valid_type | code | resp_body    |
  | 1        | 1       | valid           | 200  | case6_1      |
  | 2        | 2       | uniq            | 200  | case6_2      |
  | 3        | 3       | invalid         | 400  | Invalid JSON |
  | 4        | 0       | valid           | 404  | case6_0      |

Scenario: Delete
  When we send POST with key as "case7_<post_key>" and valid JSON as body
  And we send DELETE with key as "case7_<del_key>"
  Then the response code will be <code>
  And the response body will be contain "case7_<del_key>"
Examples:
  | post_key | del_key | code |
  | 1        | 1       | 200  |
  | 2        | 0       | 404  |


Scenario: Data was actualy deleted
  When we send POST with key as "case8" and uniq JSON as body
  And we send DELETE with key as "case8"
  And we send GET with key as "case8"
  And the response code will be 404
  And the response body will be contain "case8"

Scenario: Data was actualy resplaced
  When we send POST with key as "case9" and valid JSON as body
  And we send PUT with key as "case9" and uniq JSON as body
  And we send GET with key as "case9"
  Then the response code will be 200
  And the response body will contains the same JSON

Scenario: POST check additional error messages
  When we send POST with key as "case10_<key>" and uniq JSON as body but without the <missed_element>
  Then the response code will be 400
  And the response body will be contain "<reason>"
Examples:
  | key | missed_element | reason               |
  | 1   | key            | Key is not defined   |
  | 2   | value          | Value is not defined |