

```mermaid
classDiagram
Person: +name String
Person: -age int
Person: +getName() String
Person: +setName(String name)


```

```mermaid
classDiagram
Animal: +name String
Animal: +age int
Animal: +makeSound() void
Dog: +breed String
Dog: +bark() void
Animal <|-- Dog


```

```mermaid
classDiagram
<<interface>> Drawable
Drawable: +draw() void
Circle: +radius int
Circle: +draw() void
Rectangle: +width int
Rectangle: +height int
Rectangle: +draw() void
Drawable <|.. Circle
Drawable <|.. Rectangle


```

```mermaid
classDiagram
Car: +make String
Car: +model String
Car: +drive() void
Engine: +type String
Car "1" *-- "1" Engine: has

```

```mermaid
classDiagram
Customer: +name String
Customer: +placeOrder() void
Order: +orderId int
Order: +process() void
Customer "1" --> "*" Order: places

```

```mermaid
classDiagram
<<abstract>> Shape
Shape: +color String
Shape: +draw() void
Circle: +radius int
Circle: +draw() void
Shape <|-- Circle


```