## Tracing Support

Initial support for tracing will be implemented with [OpenTelemetry](https://opentelemetry.io). What follows is an RFC for community review.

## Considerations (aka RFC)

#### Trace Context.

There appears to be two different commonly used http trace propagation contexts. In this context, propogation refers to the http headers.

 - WC3 https://www.w3.org/TR/trace-context-1/#traceparent-header
 - OpenZipkin https://github.com/openzipkin/b3-propagation

The recommendation for Trickster is that we support both. WC3 is widely recognized and a search of 


#### 
