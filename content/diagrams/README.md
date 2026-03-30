# Diagram Examples

This page demonstrates D2 and Mermaid diagram support.

## D2 Diagram

```d2
direction: right

A: {
  shape: circle
  label: Start
}

B: {
  shape: rectangle
  label: Process
}

C: {
  shape: diamond
  label: Decision
}

A -> B
B -> C
```

## Mermaid Flowchart

```mermaid
graph TD
    A[Start] --> B{Is it working?}
    B -->|Yes| C[Great!]
    B -->|No| D[Debug]
    D --> E[Fix]
    E --> B
```

## Mermaid Sequence Diagram

```mermaid
sequenceDiagram
    participant Client
    participant Server
    participant Database

    Client->>Server: Request page
    Server->>Database: Query content
    Database-->>Server: Return content
    Server-->>Client: Render HTML
```
