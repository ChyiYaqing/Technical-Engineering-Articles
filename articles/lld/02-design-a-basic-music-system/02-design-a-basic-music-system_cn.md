---
title: "Design a basic Music System"
url: "https://x.com/Harry_The_Nerd/status/2056620076246479055"
category: "LLD"
date: "2026-05-19"
description: "Low-level design of a basic music streaming/playback system."
lang: "zh-CN"
---

# 设计一个基础的音乐系统

> 一个基础音乐流媒体/播放系统的详细设计。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2056620076246479055](https://x.com/Harry_The_Nerd/status/2056620076246479055) · 2026-05-19

![封面图](https://pbs.twimg.com/media/HIm42gEawAEqjWw.jpg)

Spotify、Apple Music、YouTube Music 这些现代音乐流媒体应用，都高度依赖结构良好的面向对象设计。这篇文章里，我们会用 Java 设计一个简化版的音乐播放列表系统，并理解背后重要的详细设计（LLD）概念。

系统支持：

- 艺人和歌曲
- 一个集中式的音乐库
- 用户创建播放列表
- 在播放列表中添加/移除歌曲
- 同一首歌被多个播放列表共享
- 对象生命周期的独立管理

**问题描述**

我们想设计这样一个音乐系统：

歌曲属于艺人。

所有歌曲存放在一个主库里。

用户可以创建多个播放列表。

同一首歌可以出现在多个播放列表中。

删除播放列表不应该删除歌曲本身。

这和真实世界里音乐应用的运作方式是一致的。

**设计中的核心类**

我们的设计包含五个主要类：Artist —— 表示一位音乐艺人；Song —— 表示一首歌；Playlist —— 保存一组歌曲；User —— 管理播放列表；Library —— 存放所有可用的歌曲。

**1\. Artist 类**

Artist 类是系统里最简单的实体之一。

```java
class Artist {
    private String name;

    public Artist(String name) {
        this.name = name;
    }

    public String getName() {
        return name;
    }
}
```

职责

- 保存艺人相关的信息
- 作为歌曲的依赖项

设计要点

这个类遵循**单一职责原则（SRP）**，因为它只管理艺人数据。

**2\. Song 类**

Song 类代表系统的核心实体。

```java
class Song {
    private String title;
    private Artist artist;
    private int duration;
}
```

每首歌包含：

- 歌曲标题
- Artist 对象
- 以秒为单位的时长

为什么用组合？—— 一首 Song 里包含一个 Artist。

这体现的是一种 **Has-A（拥有）关系**。

Song HAS-A Artist，这就是面向对象设计里的组合。

**3\. Playlist 类**

播放列表管理一组歌曲。

```java
class Playlist {
    private String name;
    private List<Song> songs = new ArrayList<>();
}
```

**功能**

- 添加歌曲
- 移除歌曲
- 统计歌曲数量
- 计算总时长
```java
public void addSong(Song song) {
    songs.add(song);
}
```

重要的详细设计概念：聚合。这个类展示的是**聚合（Aggregation）**。

播放列表并不永久拥有歌曲，歌曲可以脱离播放列表独立存在。

**Playlist HAS-A 一组 Song**

如果一个播放列表被删除：

- 歌曲依然存在
- 歌曲可能还属于其他播放列表
- 歌曲仍然留在音乐库里

这是教科书级别的聚合示例。

**4\. User 类**

用户可以创建和管理播放列表。

```java
class User {
    private String name;
    private List<Playlist> playlists = new ArrayList<>();
}
```

**职责**

- 创建播放列表
- 删除播放列表
- 保存用户的播放列表
```java
public Playlist createPlaylist(String playlistName) {
    Playlist playlist = new Playlist(playlistName);
    playlists.add(playlist);
    return playlist;
}
```

**User 与 Playlist 的关系**

用户管理播放列表，但播放列表仍然是独立的实体。这让设计保持灵活、易于扩展。

**5\. Library 类**

Library 充当歌曲的中央仓库。

```java
class Library {
    private List<Song> songs = new ArrayList<>();
}
```

**为什么需要一个 Library？**

如果没有集中式的音乐库：

- 歌曲会在各个播放列表里重复存储
- 内存占用会上升
- 歌曲管理会变得困难

有了它，播放列表只需要引用已有的歌曲。这样更省内存，也更贴近真实系统的做法。

**UML 图**

![](https://pbs.twimg.com/media/HIoppDAbkAAnsst.png)

**用到的关键详细设计概念**

**1\. 封装** 所有字段都是 private 的。

```java
private String title;
```

访问通过 getter 和方法来控制，从而保护内部状态。

**2\. 关联** 歌曲与艺人之间存在关联关系。

**3\. 聚合**

Playlist -\> Songs。播放列表使用歌曲，但不掌控它们的生命周期。

**4\. 单一职责原则**

每个类都只干一件事。

Artist —— 艺人信息

Song —— 歌曲信息

Playlist —— 播放列表操作

User —— 用户操作

Library —— 歌曲存储

**5\. 复用性**

同一个歌曲对象可以在多个播放列表之间复用。

```java
workout.addSong(hello);
chill.addSong(hello);
```

**时间复杂度分析**

向播放列表添加歌曲 —— O(1)；移除歌曲 —— O(n)；计算总时长 —— O(n)；创建播放列表 —— O(1)。

完整代码 ——

```java
import java.util.ArrayList;
import java.util.List;

class Artist {
    private final String artistName;

    public Artist(String artistName) {
        this.artistName = artistName;
    }

    public String getArtistName() {
        return artistName;
    }
}

class Song {
    private final String songTitle;
    private final Artist singer;
    private final int lengthInSeconds;

    public Song(String songTitle, Artist singer, int lengthInSeconds) {
        this.songTitle = songTitle;
        this.singer = singer;
        this.lengthInSeconds = lengthInSeconds;
    }

    public String getSongTitle() {
        return songTitle;
    }

    public Artist getSinger() {
        return singer;
    }

    public int getLengthInSeconds() {
        return lengthInSeconds;
    }

    @Override
    public String toString() {
        return songTitle + " - " + singer.getArtistName()
                + " [" + lengthInSeconds + " sec]";
    }
}

class Playlist {
    private final String playlistTitle;
    private final List<Song> trackList;

    public Playlist(String playlistTitle) {
        this.playlistTitle = playlistTitle;
        this.trackList = new ArrayList<>();
    }

    public void addTrack(Song song) {
        trackList.add(song);
    }

    public void deleteTrack(Song song) {
        trackList.remove(song);
    }

    public int totalTracks() {
        return trackList.size();
    }

    public int calculateDuration() {
        int duration = 0;

        for (Song track : trackList) {
            duration += track.getLengthInSeconds();
        }

        return duration;
    }

    public String getPlaylistTitle() {
        return playlistTitle;
    }

    public List<Song> fetchTracks() {
        return trackList;
    }
}

class User {
    private final String username;
    private final List<Playlist> userPlaylists;

    public User(String username) {
        this.username = username;
        this.userPlaylists = new ArrayList<>();
    }

    public Playlist makePlaylist(String title) {
        Playlist playlist = new Playlist(title);
        userPlaylists.add(playlist);

        return playlist;
    }

    public void removePlaylist(Playlist playlist) {
        userPlaylists.remove(playlist);
    }

    public String getUsername() {
        return username;
    }

    public List<Playlist> getUserPlaylists() {
        return userPlaylists;
    }
}

class MusicLibrary {
    private final List<Song> availableSongs;

    public MusicLibrary() {
        availableSongs = new ArrayList<>();
    }

    public void insertSong(Song song) {
        availableSongs.add(song);
    }

    public int totalSongs() {
        return availableSongs.size();
    }

    public List<Song> getAvailableSongs() {
        return availableSongs;
    }
}

public class Main {

    public static void main(String[] args) {

        Artist coldplay = new Artist("Coldplay");
        Artist adele = new Artist("Adele");

        Song yellow = new Song("Yellow", coldplay, 269);
        Song clocks = new Song("Clocks", coldplay, 307);
        Song hello = new Song("Hello", adele, 295);
        Song someoneLikeYou = new Song("Someone Like You", adele, 285);

        MusicLibrary musicLibrary = new MusicLibrary();

        musicLibrary.insertSong(yellow);
        musicLibrary.insertSong(clocks);
        musicLibrary.insertSong(hello);
        musicLibrary.insertSong(someoneLikeYou);

        User alice = new User("Alice");

        Playlist gymPlaylist = alice.makePlaylist("Workout Mix");
        Playlist relaxingPlaylist = alice.makePlaylist("Chill Vibes");

        gymPlaylist.addTrack(yellow);
        gymPlaylist.addTrack(clocks);
        gymPlaylist.addTrack(hello);

        relaxingPlaylist.addTrack(hello);
        relaxingPlaylist.addTrack(someoneLikeYou);

        System.out.println("Songs present in library : "
                + musicLibrary.totalSongs());

        System.out.println();

        System.out.println(
                gymPlaylist.getPlaylistTitle()
                        + " -> "
                        + gymPlaylist.totalTracks()
                        + " songs, Duration : "
                        + gymPlaylist.calculateDuration() + "s"
        );

        for (Song track : gymPlaylist.fetchTracks()) {
            System.out.println(" * " + track);
        }

        System.out.println();

        System.out.println(
                relaxingPlaylist.getPlaylistTitle()
                        + " -> "
                        + relaxingPlaylist.totalTracks()
                        + " songs, Duration : "
                        + relaxingPlaylist.calculateDuration() + "s"
        );

        for (Song track : relaxingPlaylist.fetchTracks()) {
            System.out.println(" * " + track);
        }

        System.out.println();

        alice.removePlaylist(gymPlaylist);

        System.out.println("After removing playlist : "
                + gymPlaylist.getPlaylistTitle());

        System.out.println("Library song count remains : "
                + musicLibrary.totalSongs());

        System.out.println("Playlist '"
                + relaxingPlaylist.getPlaylistTitle()
                + "' still contains "
                + relaxingPlaylist.totalTracks()
                + " songs");

        System.out.println("'"
                + yellow.getSongTitle()
                + "' still exists independently.");
    }
}
```

以上就是全部内容，各位，干杯！！
