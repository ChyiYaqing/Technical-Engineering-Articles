---
title: "Design Youtube"
url: "https://x.com/Harry_The_Nerd/status/2060001018134553006"
category: "LLD"
date: "2026-05-28"
description: "Low-level design walkthrough for a YouTube-like system."
lang: "zh-CN"
---

# 设计 YouTube

> 类 YouTube 系统的详细设计全流程讲解。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2060001018134553006](https://x.com/Harry_The_Nerd/status/2060001018134553006) · 2026-05-28

![封面图](https://pbs.twimg.com/media/HJaTZXfbUAA00or.jpg)

## 概述（问题描述）

设计一个像 YouTube 这样的平台，本质上是画一张干净、条理清晰的蓝图，让代码能不断生长而不会散架。观察真实用户怎么用 YouTube——订阅频道、给视频点赞、回复评论——我们就能设计出一套（至少是）聪明可靠的结构。本文拆解一个视频平台的详细设计，看看代码是怎么组织的、各部分之间如何交互，以及哪些核心设计规则让系统保持灵活。我尽量写得对新手友好，不过还是建议先读第一篇详细设计文章（打好概念基础），并在 leetcode 上做一些设计类题目。

## 架构与类关系

在写实际代码之前，先看看全局。在面向对象编程里，类之间的连接方式主要有以下几种：

- **继承（"Is-A" 关系）：** Creator **是一个** User。创作者能做普通用户能做的一切（比如看视频），此外还多了一些专属能力（比如上传视频）。
- **组合（"Part-Of" 关系）：** Video 严格属于某个 Creator。创作者删掉频道，视频也随之删除。视频不能脱离创作者而存在。
- **聚合（"Has-A" 关系）：** Creator 有一份订阅者（User）名单。但这些用户是独立存在的，创作者删掉频道，用户账号照样安然无恙。
- **组合结构（"自引用"）：** 一条 Comment 里可以包含其他 Comment 对象作为回复，从而形成层层嵌套的对话线程。

## 核心领域类

**1\. 状态配置**

一个视频在生命周期里会经历不同阶段。我们用 **Enum**（一组固定常量的特殊列表）来记录视频是仍在处理中、公开、私有还是已删除。

```java
enum VideoState {
    PUBLIC, PRIVATE, PROCESSING, DELETED
}
```

**2\. 用户**

User 类代表平台上的普通用户。它保存用户的资料（ID、姓名），并记录个人的互动行为，比如观看历史、点赞过的视频和订阅关系。

```java
class User {
    private String userId;
    private String name;
    private List<Creator> subscribedTo;
    private List<Video> watchHistory;
    private List<Video> likedVideos;
    private int watchedHours;

    public User(String userId, String name) {
        this.userId = userId;
        this.name = name;
        this.subscribedTo = new ArrayList<>();
        this.watchHistory = new ArrayList<>();
        this.likedVideos = new ArrayList<>();
        this.watchedHours = 0;
    }

    // Like a video — updates both User's likedVideos and Video's likes list
    public void likeAVideo(Video video) {
        if (!likedVideos.contains(video)) {
            likedVideos.add(video);
            video.addLike(this);
        }
    }

    // Comment on a video with a body string
    public void commentOnAVideo(Video video, String body) {
        Comment comment = new Comment(
            java.util.UUID.randomUUID().toString(), body, this
        );
        video.addComment(comment);
    }

    // Share a video: increments the share counter
    public void shareAVideo(Video video) {
        video.incrementShares();
    }

    // Subscribe: bidirectional update
    public void subscribeToCreator(Creator creator) {
        if (!subscribedTo.contains(creator)) {
            subscribedTo.add(creator);
            creator.addSubscriber(this);
        }
    }

    // Unsubscribe: bidirectional update
    public void unsubscribeFromCreator(Creator creator) {
        subscribedTo.remove(creator);
        creator.removeSubscriber(this);
    }

    public void clearHistory() {
        watchHistory.clear();
    }

    public void deleteFromHistory(Video video) {
        watchHistory.remove(video);
    }

    // Called by the system when a video is watched
    void addToHistory(Video video) {
        if (!watchHistory.contains(video)) {
            watchHistory.add(video);
        }
    }

    // Getters
    public String getUserId() { return userId; }
    public String getName() { return name; }
    public List<Video> getLikedVideos() { return likedVideos; }
    public List<Video> getWatchHistory() { return watchHistory; }
}
```

**3\. 内容创作者**

由于 Creator 继承自 User，它自动获得我们刚写好的所有普通用户能力。在此之上，它再加上创作者特有的行为，比如管理已上传视频列表、跟踪订阅者。

```java
class Creator extends User {
    private List<Video> videos;        // Composition
    private List<User> subscribers;    // Aggregation
    private double earnings;

    public Creator(String userId, String name) {
        super(userId, name);
        this.videos = new ArrayList<>();
        this.subscribers = new ArrayList<>();
        this.earnings = 0.0;
    }

    // Upload is composition: video belongs to creator
    public void uploadVideo(Video video) {
        videos.add(video);
        video.setState(VideoState.PUBLIC);
        notifySubscribers(video);   // Observer Pattern
    }

    public void deleteVideo(Video video) {
        videos.remove(video);
        video.setState(VideoState.DELETED);
    }

    // Observer Pattern, notify all subscribers
    public void notifySubscribers(Video video) {
        for (User subscriber : subscribers) {
            System.out.println(
                "Notifying " + subscriber.getName() +
                ": New video uploaded — " + video.getTitle()
            );
        }
    }

    // Called by User.subscribeToCreator()
    void addSubscriber(User user) {
        if (!subscribers.contains(user)) {
            subscribers.add(user);
        }
    }

    void removeSubscriber(User user) {
        subscribers.remove(user);
    }

    public List<Video> getVideos() { return videos; }
    public List<User> getSubscribers() { return subscribers; }
    public double getEarnings() { return earnings; }
}
```

**4\. 视频模型**

Video 类是单个视频相关所有信息的容器，包括标题、时长、作者，以及点赞和评论过它的用户列表。

```java
class Video {
    private String videoId;
    private String title;
    private String description;
    private String thumbnail;       // URL string
    private int duration;           // in seconds
    private List<String> tags;
    private VideoState state;
    private Creator creator;
    private List<User> likes;
    private int shares;
    private List<Comment> comments;
    private LocalDateTime uploadedAt;

    public Video(String videoId, String title, String description,
                 String thumbnail, int duration,
                 List<String> tags, Creator creator) {
        this.videoId = videoId;
        this.title = title;
        this.description = description;
        this.thumbnail = thumbnail;
        this.duration = duration;
        this.tags = tags;
        this.creator = creator;
        this.state = VideoState.PROCESSING;
        this.likes = new ArrayList<>();
        this.shares = 0;
        this.comments = new ArrayList<>();
        this.uploadedAt = LocalDateTime.now();
    }

    public void addLike(User user) {
        if (!likes.contains(user)) {
            likes.add(user);
        }
    }

    public void addComment(Comment comment) {
        comments.add(comment);
    }

    public void incrementShares() {
        shares++;
    }

    public void setState(VideoState state) {
        this.state = state;
    }

    // Getters
    public String getVideoId() { return videoId; }
    public String getTitle() { return title; }
    public VideoState getState() { return state; }
    public Creator getCreator() { return creator; }
    public List<User> getLikes() { return likes; }
    public List<Comment> getComments() { return comments; }
    public int getShares() { return shares; }
}
```

**5\. 评论**

在 YouTube 上，评论区不是一个扁平列表：人们会回复评论，还会回复这些回复。为了支撑这一点，每个 Comment 对象内部都包含一个 Comment 对象的列表。

```java
class Comment {
    private String commentId;
    private String body;
    private User owner;
    private List<User> likes;
    private List<Comment> replies;   // Composite Pattern
    private LocalDateTime commentedAt;

    public Comment(String commentId, String body, User owner) {
        this.commentId = commentId;
        this.body = body;
        this.owner = owner;
        this.likes = new ArrayList<>();
        this.replies = new ArrayList<>();
        this.commentedAt = LocalDateTime.now();
    }

    public void likeComment(User user) {
        if (!likes.contains(user)) {
            likes.add(user);
        }
    }

    // Composite Pattern.. add a reply (which is itself a Comment)
    public void addReply(Comment reply) {
        replies.add(reply);
    }

    // Getters
    public String getCommentId() { return commentId; }
    public String getBody() { return body; }
    public User getOwner() { return owner; }
    public List<Comment> getReplies() { return replies; }
    public List<User> getLikes() { return likes; }
    public LocalDateTime getCommentedAt() { return commentedAt; }
}
```

## SOLID 原则的运用

**单一职责原则（SRP）**：每个类只负责一件事。User 管理用户行为，Video 保存视频数据，Comment 处理评论结构，谁也不越界。

**里氏替换原则（LSP）：** Creator 继承自 User。任何期望 User 的地方都能换成 Creator，而不破坏原有行为。一个 List<User\> 类型的订阅者列表，天然可以同时装普通用户和创作者。

**开闭原则（OCP）：** 这个设计对扩展是开放的。想要 PremiumUser？继承 User。想要 LiveStream？继承 Video。现有的类一个都不用改。

**接口隔离原则（ISP）：** 我们没有把所有方法塞进一个臃肿的接口。用户和创作者各有一套与自身角色相关的方法。

## 用到的设计模式

**1\. 观察者模式：** Creator 上传视频时会触发 notifySubscribers()，把更新推送给所有订阅者。Creator = 主题（Subject），User = 观察者（Observer）。

**2\. 组合模式：** Comment 持有一个 List<Comment\> 作为回复。这种递归结构支持无限嵌套：一条评论可以有回复，回复本身又能有回复。

**3\. 工厂模式：** UserFactory 承担 User 和 Creator 的创建逻辑，把实例化逻辑挡在业务代码之外。

## 关键关系

- Creator **继承（extends）** User —— 继承
- Creator **拥有** List<Video\> —— 组合（视频不能脱离创作者存在）
- Creator **拥有** List<User\> subscribers —— 聚合（订阅者独立存在）
- Video **拥有** List<Comment\> —— 关联
- Comment **拥有** List<Comment\> replies —— 自引用（组合模式）

## 最终代码

```java
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;

// ENUM

enum VideoState {
    PUBLIC, PRIVATE, PROCESSING, DELETED
}

// USER CLASS

class User {
    private String userId;
    private String name;
    private List<Creator> subscribedTo;
    private List<Video> watchHistory;
    private List<Video> likedVideos;
    private int watchedHours;

    public User(String userId, String name) {
        this.userId = userId;
        this.name = name;
        this.subscribedTo = new ArrayList<>();
        this.watchHistory = new ArrayList<>();
        this.likedVideos = new ArrayList<>();
        this.watchedHours = 0;
    }

    // Like a video- updates both User's likedVideos and Video's likes list
    public void likeAVideo(Video video) {
        if (!likedVideos.contains(video)) {
            likedVideos.add(video);
            video.addLike(this);
        }
    }

    // Comment on a video with a body string
    public void commentOnAVideo(Video video, String body) {
        Comment comment = new Comment(
            java.util.UUID.randomUUID().toString(), body, this
        );
        video.addComment(comment);
    }

    // Share a video- increments the share counter
    public void shareAVideo(Video video) {
        video.incrementShares();
    }

    // Subscribe- bidirectional update
    public void subscribeToCreator(Creator creator) {
        if (!subscribedTo.contains(creator)) {
            subscribedTo.add(creator);
            creator.addSubscriber(this);
        }
    }

    // Unsubscribe- bidirectional update
    public void unsubscribeFromCreator(Creator creator) {
        subscribedTo.remove(creator);
        creator.removeSubscriber(this);
    }

    public void clearHistory() {
        watchHistory.clear();
    }

    public void deleteFromHistory(Video video) {
        watchHistory.remove(video);
    }

    // Called by the system when a video is watched
    void addToHistory(Video video) {
        if (!watchHistory.contains(video)) {
            watchHistory.add(video);
        }
    }

    // Getters
    public String getUserId() { return userId; }
    public String getName() { return name; }
    public List<Video> getLikedVideos() { return likedVideos; }
    public List<Video> getWatchHistory() { return watchHistory; }
}

// CREATOR CLASS extends User
// LSP: Creator is-a User
// Observer Pattern: Creator is the Subject

class Creator extends User {
    private List<Video> videos;        // Composition
    private List<User> subscribers;    // Aggregation
    private double earnings;

    public Creator(String userId, String name) {
        super(userId, name);
        this.videos = new ArrayList<>();
        this.subscribers = new ArrayList<>();
        this.earnings = 0.0;
    }

    // Upload is composition: video belongs to creator
    public void uploadVideo(Video video) {
        videos.add(video);
        video.setState(VideoState.PUBLIC);
        notifySubscribers(video);   // Observer Pattern
    }

    public void deleteVideo(Video video) {
        videos.remove(video);
        video.setState(VideoState.DELETED);
    }

    // Observer Pattern - notify all subscribers
    public void notifySubscribers(Video video) {
        for (User subscriber : subscribers) {
            System.out.println(
                "Notifying " + subscriber.getName() +
                ": New video uploaded - " + video.getTitle()
            );
        }
    }

    // Called by User.subscribeToCreator()
    void addSubscriber(User user) {
        if (!subscribers.contains(user)) {
            subscribers.add(user);
        }
    }

    void removeSubscriber(User user) {
        subscribers.remove(user);
    }

    public List<Video> getVideos() { return videos; }
    public List<User> getSubscribers() { return subscribers; }
    public double getEarnings() { return earnings; }
}

// VIDEO CLASS

class Video {
    private String videoId;
    private String title;
    private String description;
    private String thumbnail;       // URL string
    private int duration;           // in seconds
    private List<String> tags;
    private VideoState state;
    private Creator creator;
    private List<User> likes;
    private int shares;
    private List<Comment> comments;
    private LocalDateTime uploadedAt;

    public Video(String videoId, String title, String description,
                 String thumbnail, int duration,
                 List<String> tags, Creator creator) {
        this.videoId = videoId;
        this.title = title;
        this.description = description;
        this.thumbnail = thumbnail;
        this.duration = duration;
        this.tags = tags;
        this.creator = creator;
        this.state = VideoState.PROCESSING;
        this.likes = new ArrayList<>();
        this.shares = 0;
        this.comments = new ArrayList<>();
        this.uploadedAt = LocalDateTime.now();
    }

    public void addLike(User user) {
        if (!likes.contains(user)) {
            likes.add(user);
        }
    }

    public void addComment(Comment comment) {
        comments.add(comment);
    }

    public void incrementShares() {
        shares++;
    }

    public void setState(VideoState state) {
        this.state = state;
    }

    // Getters
    public String getVideoId() { return videoId; }
    public String getTitle() { return title; }
    public VideoState getState() { return state; }
    public Creator getCreator() { return creator; }
    public List<User> getLikes() { return likes; }
    public List<Comment> getComments() { return comments; }
    public int getShares() { return shares; }
}

// COMMENT CLASS
// Composite Pattern: Comment has List<Comment>

class Comment {
    private String commentId;
    private String body;
    private User owner;
    private List<User> likes;
    private List<Comment> replies;   // Composite Pattern
    private LocalDateTime commentedAt;

    public Comment(String commentId, String body, User owner) {
        this.commentId = commentId;
        this.body = body;
        this.owner = owner;
        this.likes = new ArrayList<>();
        this.replies = new ArrayList<>();
        this.commentedAt = LocalDateTime.now();
    }

    public void likeComment(User user) {
        if (!likes.contains(user)) {
            likes.add(user);
        }
    }

    // Composite Pattern- add a reply (which is itself a Comment)
    public void addReply(Comment reply) {
        replies.add(reply);
    }

    // Getters
    public String getCommentId() { return commentId; }
    public String getBody() { return body; }
    public User getOwner() { return owner; }
    public List<Comment> getReplies() { return replies; }
    public List<User> getLikes() { return likes; }
    public LocalDateTime getCommentedAt() { return commentedAt; }
}

// FACTORY PATTERN — UserFactory

class UserFactory {
    public static User createUser(String userId, String name) {
        return new User(userId, name);
    }

    public static Creator createCreator(String userId, String name) {
        return new Creator(userId, name);
    }
}

// MAIN — Demo

public class YouTubeLLD {
    public static void main(String[] args) {

        // Factory creates objects
        Creator creator = UserFactory.createCreator("c1", "TechWithTim");
        User user1 = UserFactory.createUser("u1", "Alice");
        User user2 = UserFactory.createUser("u2", "Bob");

        // User subscribes to creator- Observer registered
        user1.subscribeToCreator(creator);
        user2.subscribeToCreator(creator);

        // Creator uploads video- Observer notified
        Video video = new Video(
            "v1", "LLD of YouTube", "Full LLD walkthrough",
            "https://thumbnail.url/lld.jpg", 1800,
            List.of("lld", "design", "java"), creator
        );
        creator.uploadVideo(video);
        // Console: Notifying Alice: New video uploaded- LLD of YouTube
        // Console: Notifying Bob:   New video uploaded- LLD of YouTube

        // Users interact with the video
        user1.likeAVideo(video);
        user1.commentOnAVideo(video, "Great explanation!");
        user2.shareAVideo(video);

        // Nested comment is a Composite Pattern
        Comment parentComment = video.getComments().get(0);
        Comment reply = new Comment("c2", "Totally agree!", user2);
        parentComment.addReply(reply);

        // Watch history, system adds automatically
        user1.addToHistory(video);

        System.out.println("Likes: " + video.getLikes().size());       // 1
        System.out.println("Shares: " + video.getShares());            // 1
        System.out.println("Comments: " + video.getComments().size()); // 1
        System.out.println("Replies on comment: " +
            parentComment.getReplies().size());                         // 1
        System.out.println("Watch history size: " +
            user1.getWatchHistory().size());                            // 1
    }
}
```

以上就是全部内容，各位，干杯！！！
