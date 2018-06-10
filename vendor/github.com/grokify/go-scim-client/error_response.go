/*
 * SCIM API
 *
 * SCIM V2 API implemented by RingCentral
 *
 * API version: 0.1.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package scim

type ErrorResponse struct {

	// detail error message
	Detail string `json:"detail,omitempty"`

	Schemas []string `json:"schemas,omitempty"`

	// bad request type when status code is 400
	ScimType string `json:"scimType,omitempty"`

	// same as HTTP status code, e.g. 400, 401, etc.
	Status string `json:"status,omitempty"`
}