---
title: "Design a Vending Machine"
url: "https://x.com/Harry_The_Nerd/status/2060348729929175403"
category: "LLD"
date: "2026-05-29"
description: "Classic LLD problem: designing a vending machine."
lang: "zh-CN"
---

# 设计一台自动售货机

> 经典的详细设计题目：设计一台自动售货机。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2060348729929175403](https://x.com/Harry_The_Nerd/status/2060348729929175403) · 2026-05-29

![封面图](https://pbs.twimg.com/media/HJfQIb2asAAVd2j.jpg)

## 概览（问题描述）

自动售货机表面上看很简单：投币、按按钮、拿零食。但真坐下来设计时你会发现，它有一大堆活动部件（叹气）。怎么给机器在不同时刻的行为建模，大概是这里最棘手的部分。

如果我们到处都用 if/else 判断（「如果投了钱、如果选了商品、如果余额够……」），VendingMachine 类会飞快地变成一团乱麻。干净的解法是**状态模式（State Pattern）**：把机器生命周期的每个阶段做成一个独立的类，各自管好自己的规则。

## 实体与类

**1\. Item（基类 + 子类型）**

很多人第一反应是分别写 Chips、Drink、Chocolate 三个类。但仔细看，它们的属性完全一样：名称、数量、价格。写三个一模一样的类就是复制粘贴代码，违反了 **DRY 原则**（Don't Repeat Yourself，不要重复自己）。

所以，写一个抽象的 Item 基类装下所有公共属性，让 Chips、Drink、Chocolate 去继承它。这样以后想加 Candy 这类新商品，只要继承 Item 就行，不用碰任何已有代码。这符合**开闭原则**（对扩展开放，对修改关闭）。

```java
abstract class Item {
    private String name;
    private int quantity;
    private int price;

    public Item(String name, int quantity, int price) {
        this.name = name;
        this.quantity = quantity;
        this.price = price;
    }

    public String getName() { return name; }
    public int getQuantity() { return quantity; }
    public int getPrice() { return price; }
    public void setQuantity(int quantity) { this.quantity = quantity; }
}

class Chips extends Item {
    public Chips(String name, int quantity, int price) { super(name, quantity, price); }
}

class Drink extends Item {
    public Drink(String name, int quantity, int price) { super(name, quantity, price); }
}

class Chocolate extends Item {
    public Chocolate(String name, int quantity, int price) { super(name, quantity, price); }
}
```

**状态模式是什么？为什么要用它？**

在进入各个状态之前，先搞清楚我们为什么需要这个模式。

设想一下不用它来写 VendingMachine。selectProduct() 方法大概会长这样：

```java
public void selectProduct(String itemName) {
    if (moneyInserted) {
        if (inventoryHasItem) {
            if (balanceSufficient) {
                // dispense
            } else {
                // error
            }
        } else {
            // error
        }
    } else {
        // error
    }
}
```

而且每一个方法（insertMoney、dispense、returnChange）都会有同样的嵌套烂摊子。加一个新状态或者改一处行为，就得在这些 if/else 块里翻来翻去。

**状态模式**的主张是：与其用一个巨大的类处理所有事情，不如给每个状态一个自己的类。每个状态只知道在自己这个状态下什么是合法的。机器只管把事情交给当前状态处理，一个 if/else 都不需要。

我们这台自动售货机的四个状态：

IDLE -\> MONEY\_INSERTED -\> DISPENSING -\> RETURNING\_CHANGE -\> IDLE

所有状态实现同一个接口，这样机器就能用同一种方式跟它们中的任何一个打交道：

```java
interface VendingMachineState {
    void insertMoney(int amount);
    void selectProduct(String itemName);
    void dispense();
    void returnChange();
}
```

**2\. IdleState**

这是默认状态，机器就在那儿闲着等。这里唯一合法的动作是投币。其他动作都应该抛错，因为什么都还没发生，选商品或者要找零都说不通。

投币之后，我们切换到 MoneyInsertedState。

```java
class IdleState implements VendingMachineState {
    private VendingMachine machine;

    public IdleState(VendingMachine machine) { this.machine = machine; }

    @Override
    public void insertMoney(int amount) {
        machine.setBalance(amount);
        machine.setState(machine.getMoneyInsertedState()); // transition
        System.out.println("Money inserted: " + amount);
    }

    @Override
    public void selectProduct(String itemName) {
        throw new RuntimeException("Please insert money first!");
    }

    @Override
    public void dispense() {
        throw new RuntimeException("Please insert money first!");
    }

    @Override
    public void returnChange() {
        throw new RuntimeException("No money to return!");
    }
}
```

**3\. MoneyInsertedState**

钱已经进机器了。现在有两件事是合法的：用户可以继续投钱（所以我们是往余额上累加，而不是覆盖），或者选一个商品。

在放行选商品之前，我们做两项检查：

- 这个商品在库存里真的存在吗？
- 余额够不够付它的价钱？

两项都通过，我们就把选中的商品记在机器上（这样 DispensingState 稍后才知道要出什么货），然后切换到 DispensingState。这里并不真的出货，因为选择和出货是两份不同的职责。

```java
class MoneyInsertedState implements VendingMachineState {
    private VendingMachine machine;

    public MoneyInsertedState(VendingMachine machine) { this.machine = machine; }

    @Override
    public void insertMoney(int amount) {
        machine.setBalance(machine.getBalance() + amount); // add, don't replace
        System.out.println("Added more money. Total: " + machine.getBalance());
    }

    @Override
    public void selectProduct(String itemName) {
        Item item = machine.getInventory().get(itemName);

        if (item == null) throw new RuntimeException("Item not found!");

        if (machine.getBalance() < item.getPrice()) {
            throw new RuntimeException("Insufficient balance! Need: "
                + item.getPrice() + " Have: " + machine.getBalance());
        }

        machine.setSelectedItem(item); // pass context to next state
        machine.setState(machine.getDispensingState());
        System.out.println("Product selected: " + itemName);
    }

    @Override
    public void dispense() { throw new RuntimeException("Select a product first!"); }

    @Override
    public void returnChange() { throw new RuntimeException("Select a product first!"); }
}
```

**4\. DispensingState**

真正干活的地方。可以想象成机器里的马达转起来，把商品实实在在推出来。

我们从余额里扣掉商品的价格，把它在库存里的数量减一，如果数量归零就整个移除。然后决定下一个状态：还有余额剩下，就去 ReturningChangeState；余额正好是零，就直接回到 IdleState。

```java
class DispensingState implements VendingMachineState {
    private VendingMachine machine;

    public DispensingState(VendingMachine machine) { this.machine = machine; }

    @Override
    public void dispense() {
        Item item = machine.getSelectedItem();

        machine.setBalance(machine.getBalance() - item.getPrice());

        item.setQuantity(item.getQuantity() - 1);
        if (item.getQuantity() == 0) {
            machine.getInventory().remove(item.getName());
        }

        System.out.println("Dispensing: " + item.getName());

        if (machine.getBalance() > 0) {
            machine.setState(machine.getReturningChangeState());
        } else {
            machine.setState(machine.getIdleState());
        }
    }

    @Override
    public void insertMoney(int amount) {
        throw new RuntimeException("Please wait, dispensing in progress!");
    }

    @Override
    public void selectProduct(String itemName) {
        throw new RuntimeException("Already dispensing!");
    }

    @Override
    public void returnChange() {
        throw new RuntimeException("Wait for dispensing to complete!");
    }
}
```

**5\. ReturningChangeState**

最简单的状态。商品已经出货了，但还有钱剩下。我们只要打印出退回多少零钱，把余额清零，然后回到 IdleState。机器就准备好迎接下一位顾客了。

```java
class ReturningChangeState implements VendingMachineState {
    private VendingMachine machine;

    public ReturningChangeState(VendingMachine machine) { this.machine = machine; }

    @Override
    public void returnChange() {
        System.out.println("Returning change: " + machine.getBalance());
        machine.setBalance(0);
        machine.setState(machine.getIdleState()); // back to start
    }

    @Override
    public void insertMoney(int amount) {
        throw new RuntimeException("Please collect your change first!");
    }

    @Override
    public void selectProduct(String itemName) {
        throw new RuntimeException("Please collect your change first!");
    }

    @Override
    public void dispense() {
        throw new RuntimeException("Already dispensed!");
    }
}
```

**6\. VendingMachine**

VendingMachine 这个类本身干净得出人意料，里面一个 if/else 都没有。它只是持有四个状态对象的引用，记录当前状态，然后把每个动作都委托给当前生效的那个状态。

它同时还持有库存（用 Map<String, Item\> 做 O(1) 查找）、当前余额，以及选中的商品（状态之间共享的上下文）。

```java
class VendingMachine {
    private Map<String, Item> inventory;
    private int balance;
    private Item selectedItem;

    private VendingMachineState idleState;
    private VendingMachineState moneyInsertedState;
    private VendingMachineState dispensingState;
    private VendingMachineState returningChangeState;
    private VendingMachineState currentState;

    public VendingMachine() {
        inventory = new HashMap<>();
        balance = 0;

        idleState = new IdleState(this);
        moneyInsertedState = new MoneyInsertedState(this);
        dispensingState = new DispensingState(this);
        returningChangeState = new ReturningChangeState(this);

        currentState = idleState; // always start idle
    }

    // delegates to current state - no if/else anywhere
    public void insertMoney(int amount) { currentState.insertMoney(amount); }
    public void selectProduct(String name) { currentState.selectProduct(name); }
    public void dispense() { currentState.dispense(); }
    public void returnChange() { currentState.returnChange(); }

    public void addItem(Item item) { inventory.put(item.getName(), item); }

    public Map<String, Item> getInventory() { return inventory; }
    public int getBalance() { return balance; }
    public void setBalance(int balance) { this.balance = balance; }
    public Item getSelectedItem() { return selectedItem; }
    public void setSelectedItem(Item item) { this.selectedItem = item; }
    public void setState(VendingMachineState state) { this.currentState = state; }
    public VendingMachineState getIdleState() { return idleState; }
    public VendingMachineState getMoneyInsertedState() { return moneyInsertedState; }
    public VendingMachineState getDispensingState() { return dispensingState; }
    public VendingMachineState getReturningChangeState() { return returningChangeState; }
}
```

**Main**

```java
import java.util.*;

// ITEMS 

abstract class Item {
    private String name;
    private int quantity;
    private int price;

    public Item(String name, int quantity, int price) {
        this.name = name;
        this.quantity = quantity;
        this.price = price;
    }

    public String getName() { return name; }
    public int getQuantity() { return quantity; }
    public int getPrice() { return price; }
    public void setQuantity(int quantity) { this.quantity = quantity; }
}

class Chips extends Item {
    public Chips(String name, int quantity, int price) {
        super(name, quantity, price);
    }
}

class Drink extends Item {
    public Drink(String name, int quantity, int price) {
        super(name, quantity, price);
    }
}

class Chocolate extends Item {
    public Chocolate(String name, int quantity, int price) {
        super(name, quantity, price);
    }
}

// STATE INTERFACE 

interface VendingMachineState {
    void insertMoney(int amount);
    void selectProduct(String itemName);
    void dispense();
    void returnChange();
}

//  IDLE STATE 

class IdleState implements VendingMachineState {
    private VendingMachine machine;

    public IdleState(VendingMachine machine) {
        this.machine = machine;
    }

    @Override
    public void insertMoney(int amount) {
        machine.setBalance(amount);
        machine.setState(machine.getMoneyInsertedState());
        System.out.println("Money inserted: " + amount);
    }

    @Override
    public void selectProduct(String itemName) {
        throw new RuntimeException("Please insert money first!");
    }

    @Override
    public void dispense() {
        throw new RuntimeException("Please insert money first!");
    }

    @Override
    public void returnChange() {
        throw new RuntimeException("No money to return!");
    }
}

// MONEY INSERTED STATE

class MoneyInsertedState implements VendingMachineState {
    private VendingMachine machine;

    public MoneyInsertedState(VendingMachine machine) {
        this.machine = machine;
    }

    @Override
    public void insertMoney(int amount) {
        machine.setBalance(machine.getBalance() + amount);
        System.out.println("Added more money. Total: " + machine.getBalance());
    }

    @Override
    public void selectProduct(String itemName) {
        Item item = machine.getInventory().get(itemName);

        if (item == null) throw new RuntimeException("Item not found!");

        if (machine.getBalance() < item.getPrice()) {
            throw new RuntimeException("Insufficient balance! Need: "
                + item.getPrice() + " Have: " + machine.getBalance());
        }

        machine.setSelectedItem(item);
        machine.setState(machine.getDispensingState());
        System.out.println("Product selected: " + itemName);
    }

    @Override
    public void dispense() {
        throw new RuntimeException("Select a product first!");
    }

    @Override
    public void returnChange() {
        throw new RuntimeException("Select a product first!");
    }
}

// DISPENSING STATE 

class DispensingState implements VendingMachineState {
    private VendingMachine machine;

    public DispensingState(VendingMachine machine) {
        this.machine = machine;
    }

    @Override
    public void dispense() {
        Item item = machine.getSelectedItem();

        machine.setBalance(machine.getBalance() - item.getPrice());

        item.setQuantity(item.getQuantity() - 1);
        if (item.getQuantity() == 0) {
            machine.getInventory().remove(item.getName());
        }

        System.out.println("Dispensing: " + item.getName());

        if (machine.getBalance() > 0) {
            machine.setState(machine.getReturningChangeState());
        } else {
            machine.setState(machine.getIdleState());
        }
    }

    @Override
    public void insertMoney(int amount) {
        throw new RuntimeException("Please wait, dispensing in progress!");
    }

    @Override
    public void selectProduct(String itemName) {
        throw new RuntimeException("Already dispensing!");
    }

    @Override
    public void returnChange() {
        throw new RuntimeException("Wait for dispensing to complete!");
    }
}

// RETURNING CHANGE STATE

class ReturningChangeState implements VendingMachineState {
    private VendingMachine machine;

    public ReturningChangeState(VendingMachine machine) {
        this.machine = machine;
    }

    @Override
    public void returnChange() {
        System.out.println("Returning change: " + machine.getBalance());
        machine.setBalance(0);
        machine.setState(machine.getIdleState());
    }

    @Override
    public void insertMoney(int amount) {
        throw new RuntimeException("Please collect your change first!");
    }

    @Override
    public void selectProduct(String itemName) {
        throw new RuntimeException("Please collect your change first!");
    }

    @Override
    public void dispense() {
        throw new RuntimeException("Already dispensed!");
    }
}

//  VENDING MACHINE 
class VendingMachine {
    private Map<String, Item> inventory;
    private int balance;
    private Item selectedItem;

    private VendingMachineState idleState;
    private VendingMachineState moneyInsertedState;
    private VendingMachineState dispensingState;
    private VendingMachineState returningChangeState;

    private VendingMachineState currentState;

    public VendingMachine() {
        inventory = new HashMap<>();
        balance = 0;

        idleState = new IdleState(this);
        moneyInsertedState = new MoneyInsertedState(this);
        dispensingState = new DispensingState(this);
        returningChangeState = new ReturningChangeState(this);

        currentState = idleState;
    }

    public void insertMoney(int amount) { currentState.insertMoney(amount); }
    public void selectProduct(String name) { currentState.selectProduct(name); }
    public void dispense() { currentState.dispense(); }
    public void returnChange() { currentState.returnChange(); }

    public void addItem(Item item) { inventory.put(item.getName(), item); }

    public Map<String, Item> getInventory() { return inventory; }
    public int getBalance() { return balance; }
    public void setBalance(int balance) { this.balance = balance; }
    public Item getSelectedItem() { return selectedItem; }
    public void setSelectedItem(Item item) { this.selectedItem = item; }
    public void setState(VendingMachineState state) { this.currentState = state; }
    public VendingMachineState getIdleState() { return idleState; }
    public VendingMachineState getMoneyInsertedState() { return moneyInsertedState; }
    public VendingMachineState getDispensingState() { return dispensingState; }
    public VendingMachineState getReturningChangeState() { return returningChangeState; }
}

// MAIN 

public class Main {
    public static void main(String[] args) {
        VendingMachine vm = new VendingMachine();

        vm.addItem(new Chips("Lays", 3, 20));
        vm.addItem(new Drink("Coke", 2, 50));
        vm.addItem(new Chocolate("KitKat", 1, 30));

        System.out.println("--- Normal Flow ---");
        vm.insertMoney(60);
        vm.selectProduct("Coke");
        vm.dispense();
        vm.returnChange();

        System.out.println("\n--- Insufficient Balance ---");
        try {
            vm.insertMoney(10);
            vm.selectProduct("Coke");
        } catch (RuntimeException e) {
            System.out.println("Error: " + e.getMessage());
        }

        System.out.println("\n--- No Money Inserted ---");
        try {
            vm.selectProduct("Lays");
        } catch (RuntimeException e) {
            System.out.println("Error: " + e.getMessage());
        }
    }
}
```

## 输出

```
--- Normal Flow ---
Money inserted: 60
Product selected: Coke
Dispensing: Coke
Returning change: 10

--- Insufficient Balance ---
Added more money. Total: 10
Error: Insufficient balance! Need: 50 Have: 10

--- No Money Inserted ---
Error: Please insert money first!
```

## 关键设计决策

**Item 基类**：所有商品的结构都一样，没必要写三个独立的类。要加新的商品类型，继承 Item 就够了。

**Map<String, Item\>**：用于库存。用 map 按名称查商品是 O(1)，换成 list 就意味着每次都要扫一遍所有商品。

**状态模式：**把行为干净利落地切分到各个状态里。VendingMachine 里没有庞大的 if/else。加一个新状态是加一个新类，而不是去改已有的类。

**selectedItem 放在 VendingMachine 上**：**MoneyInsertedState** 选定商品，**DispensingState** 使用它。把它存在机器上，就是我们在状态之间传递这份上下文的方式。

**dispense 与 selectProduct 分开：**选择和实际出货是两个不同的动作。把它们放在不同的状态里，流程更清晰，也更贴近现实。

以上就是全部内容，各位……干杯！！
