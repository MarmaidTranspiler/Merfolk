

```mermaid
    classDiagram
    <<interface>> Flyable
    Flyable : +void fly()
    
    <<interface>> Swimable
    Swimable : +void swim()
    
    class Duck {
        +void quack()
        +void swim()
        +void fly()
    }
    
    Duck ..|> Flyable
    Duck ..|> Swimable
```
