## Tracing Support

Initial support for tracing will be implemented with [OpenTelemetry](https://opentelemetry.io). What follows is an RFC for community review.

### Considerations

#### Interface 

For good reason, it has been suggested that we interface the internal tracing. Rather than use a custom adapter for use with various tracing frameworks, I suggest we start with OpenTelemetry and adapt other frameworks to fit its interface. 

#### Trace Context

There appears to be two different commonly used http trace propagation contexts. In this context, propogation refers to the http headers.

 - WC3 https://www.w3.org/TR/trace-context-1/#traceparent-header
 - OpenZipkin https://github.com/openzipkin/b3-propagation

*Recommendation*

Support both
