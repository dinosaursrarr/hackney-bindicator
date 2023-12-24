# Hackney Bindicator

## Introduction

This API tells you what bins to put out on what days for properties in the London Borough of Hackney.

[Hackney Council](http://www.hackney.gov.uk) provide a [web tool](http://hackney-waste-pages.azurewebsites.net) to access this information, but it is difficult to use programatically because it:

- involves making multiple calls to their backend;
- provides verbose output closely coupled to the Council's operational database; and
- is poorly documented.

This API is a simple layer on top of the Council's API that makes all the necessary calls for you and provides a simple JSON response. It employs caching to provide a faster response than orchestrating these calls yourself.

## API format

The API provides two endpoints, `addresses` and `property`.

### Addresses

The `/addresses/{postcode}` endpoint provides a list of properties and their IDs within a given postcode. The returned IDs are needed as input for the `property` endpoint. It returns a 400 error for invalid postcodes or postcodes outside Hackney, or a 500 error if something else went wrong.

The output format for a given postcode is:
```json
[
  {
    "Id": "foo",
    "Name": "29 ACACIA AVENUE"
  },
  {
    "Id": "bar",
    "Name": "52 FESTIVE ROAD"
  }
]
```

### Property

The `/property/{property_id}` endpoint provides a list of bins at a given property and the next collection date for each. It returns a 400 error for an unrecognized property ID, or a 500 error if something else went wrong.

The output format for a given property ID is:
```json
{
  "PropertyId": "foo",
  "Bins": [
    {
      "Name": "Garbage can",
      "Type": "rubbish",
      "NextCollection": "2024-01-01T00:00:00Z"
    },
    {
      "Name": "Recycling sack",
      "Type": "recycling",
      "NextCollection": "2024-01-05T00:00:00Z"
    }
  ]
}
```

All collection dates are truncated to the start of the relevant day (the time part is always `00:00:00`). Bin names are passed through from the Council's API. I believe there is a finite set, but am not confident I have seen all the values yet. The values seen to date are translated to one of these types: `food`, `recycling`, `garden` and `rubbish` (otherwise `unknown`).

## Use case

I made this so that I could create a [Tidbyt](http://tidbyt.com) app to show me what bins to put out after moving back to Hackney. Without this API layer, the app would have timed out. Using the API is faster as it can parallelise calls to the Council's API and cache responses.

## Source code

Source code is available on [Github](https://github.com/dinosaursrarr/hackney-bindicator)

You can [email me](mailto:tom+bindicator@chamberlaincurtis.com) if you have any questions. I can't promise a response.

## Disclaimer

I am in no way affiliated with the London Borough of Hackney Council except for being a resident. This API could break at any time if the Council update their web tool. I hope they won't because it is useful.