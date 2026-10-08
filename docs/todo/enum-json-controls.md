# ADR: Add JSON controls for TGo enums

Status: proposed

## Context

`encoding/json` can encode normal Go structs. Struct field tags already control field names,
omitted values, and other field rules. TGo must keep that behavior.

A TGo enum has a private representation. Without generated JSON methods, `encoding/json`
cannot encode its variant and payload. Services also use different enum JSON forms. The source
must state the form without a general derive system.

## Decision

Generate `MarshalJSON` and `UnmarshalJSON` for every TGo enum. Do not add `derive`, codec traits,
or a codec registry. Normal structs continue to use `encoding/json` directly.

An optional tag after `enum` selects the representation. A tag after a variant payload changes
the variant name. Payload fields use normal Go `json` tags.

```text
type Event enum `json:"adjacent,tag=type,content=data"` {
    AccountCreated struct {
        AccountID string `json:"account_id"`
    } `json:"account_created"`
    AccountClosed struct {
        AccountID string `json:"account_id"`
        Reason string `json:"reason,omitempty"`
    } `json:"account_closed"`
}
```

The default variant name is its TGo name. An explicit variant name must be unique and not empty.
Existing Go field tag rules control payload field names.

## Representations

The default is external tagging:

```text
type Event enum `json:"external"` { ... }
{"account_created":{"account_id":"a1"}}
```

Internal tagging puts the variant name in the payload object:

```text
type Event enum `json:"internal,tag=type"` { ... }
{"type":"account_created","account_id":"a1"}
```

Adjacent tagging keeps the tag and payload in separate fields:

```text
type Event enum `json:"adjacent,tag=type,content=data"` { ... }
{"type":"account_created","data":{"account_id":"a1"}}
```

Untagged encoding writes only the payload object:

```text
type Event enum `json:"untagged"` { ... }
{"account_id":"a1"}
```

`tag` is required for internal and adjacent tagging. `content` is required only for adjacent
tagging. The tag and content names must differ. An internal tag name must not conflict with a
payload field JSON name.

For untagged decoding, the decoder tries variants in declaration order. The first successful
payload decode wins. Variant order is therefore part of the wire contract. This form can be
ambiguous when two payloads accept the same object.

## Generated behavior

`MarshalJSON` uses an exhaustive switch on the enum tag. It reads the matching payload and gives
the selected wire value to `encoding/json`. It returns an error for tag zero or an unknown tag.

`UnmarshalJSON` reads the selected representation, rejects an unknown variant name, and decodes
the payload with `encoding/json`. It calls the generated variant constructor. It assigns the
receiver only after all work succeeds. Thus every successful decode contains one known variant.

The generated methods change no enum layout. They add work only when code uses JSON. Go callers
continue to use `json.Marshal` and `json.Unmarshal`.

## Compatibility

Enum and variant tags are wire contracts. A representation change, tag name change, content name
change, variant rename, or variant reorder for an untagged enum can break stored data and clients.

Payload fields keep normal Go JSON behavior. Their names, `omitempty` rules, custom JSON methods,
and errors come from `encoding/json`.

## Drawbacks

Generated methods add code for enums that never use JSON. Untagged decoding can hide an ambiguous
wire value by selecting the first matching variant. Internal tagging cannot use a payload field
with the same JSON name as its tag field.

## Scope

This decision covers JSON for TGo enums. It does not add a general derive system. It does not
change JSON behavior for normal Go structs or define support for another codec.
