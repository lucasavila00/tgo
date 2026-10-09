# Proposed language changes

These ADRs describe two small changes that remove repeated failure branches:

- [Pass errors without a wrapper](transparent-error-propagation.md)
- [Propagate Boolean absence](boolean-presence-propagation.md)

Both changes use postfix `?`. The first ADR defines the operator. The second ADR extends it to
functions that use a final `bool` result to report presence.
