package errors

import (
	"itv/shared/response"
)

var (
	BindingJSONError = &response.Response{
		Success: false,
		Errors: []*response.Error{
			{
				Message: "binding JSON error",
			},
		},
	}

	RequestCreationError = &response.Response{
		Success: false,
		Errors: []*response.Error{
			{
				Message: "something went wrong during creation request",
			},
		},
	}

	SendRequestError = &response.Response{
		Success: false,
		Errors: []*response.Error{
			{
				Message: "something went wrong during sending request",
			},
		},
	}

	StoringToDBError = &response.Response{
		Success: false,
		Errors: []*response.Error{
			{
				Message: "something went wrong during storing request's result",
			},
		},
	}

	QueryNotFoundInDB = &response.Response{
		Success: false,
		Errors: []*response.Error{
			{
				Message: "requested query not found",
			},
		},
	}
)
