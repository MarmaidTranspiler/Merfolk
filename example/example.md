

```mermaid
classDiagram
Animal <|-- Dog
Animal : -eat(Food food)
Dog : -bark(String message)
Dog "1" -- "*" Bone
Bone : -int size


```




