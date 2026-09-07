---
title: "Design a basic Music System"
url: "https://x.com/Harry_The_Nerd/status/2056620076246479055"
category: "LLD"
date: "2026-05-19"
description: "Low-level design of a basic music streaming/playback system."
---

# Design a basic Music System

> Low-level design of a basic music streaming/playback system.
>
> 原文：[https://x.com/Harry_The_Nerd/status/2056620076246479055](https://x.com/Harry_The_Nerd/status/2056620076246479055) · 2026-05-19

![Cover image](https://pbs.twimg.com/media/HIm42gEawAEqjWw.jpg)

Modern music streaming applications such as Spotify, Apple Music, or YouTube Music rely heavily on well-structured object-oriented design. In this article, we will design a simplified Music Playlist System using Java and understand the important Low-Level Design (LLD) concepts behind it.

The system supports:

- Artists and songs
- A centralized music library
- Users creating playlists
- Adding/removing songs from playlists
- Shared songs across multiple playlists
- Independent lifecycle management of objects

**Problem Statement**

We want to design a music system where:

Songs belong to artists.

All songs are stored in a master library.

Users can create multiple playlists.

A single song can exist in multiple playlists.

Deleting a playlist should not delete the songs themselves.

This mirrors how real-world music applications work.

**Core Classes in the Design**

Our design consists of five primary classes: Artist - Represents a music artist Song - Represents a song Playlist - Stores a collection of songs User - Manages playlists Library - Stores all available songs

**1\. Artist Class**

The Artist class is one of the simplest entities in the system.

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

Responsibility

- Stores artist-related information
- Acts as a dependency for songs

Design Insight

This class follows the **Single Responsibility Principle (SRP)** because it only manages artist data.

**2\. Song Class**

The Song class represents the core entity of the system.

```java
class Song {
    private String title;
    private Artist artist;
    private int duration;
}
```

Each song contains:

- Song title
- Artist object
- Duration in seconds

Why Composition? - A Song contains an Artist.

This represents a **Has-A relationship**.

Song HAS-A Artist This is composition in object-oriented design.

**3\. Playlist Class**

The playlist manages a collection of songs.

```java
class Playlist {
    private String name;
    private List<Song> songs = new ArrayList<>();
}
```

**Features**

- Add songs
- Remove songs
- Count songs
- Calculate total duration
```java
public void addSong(Song song) {
    songs.add(song);
}
```

Important LLD Concept: Aggregation This class demonstrates **Aggregation**.

A playlist does not own songs permanently. Songs can exist independently outside the playlist.

**Playlist HAS-A collection of Songs**

If a playlist gets deleted:

- Songs still exist
- Songs may still belong to other playlists
- Songs remain in the library

This is a textbook example of aggregation.

**4\. User Class**

A user can create and manage playlists.

```java
class User {
    private String name;
    private List<Playlist> playlists = new ArrayList<>();
}
```

**Responsibilities**

- Create playlists
- Delete playlists
- Store user playlists
```java
public Playlist createPlaylist(String playlistName) {
    Playlist playlist = new Playlist(playlistName);
    playlists.add(playlist);
    return playlist;
}
```

**Relationship Between User and Playlist**

A user manages playlists, but the playlists remain independent entities. This keeps the design flexible and scalable.

**5\. Library Class**

The Library serves as the central repository for songs.

```java
class Library {
    private List<Song> songs = new ArrayList<>();
}
```

**Why Do We Need a Library?**

Without a centralized library:

- Songs would be duplicated across playlists
- Memory usage would increase
- Song management becomes difficult

Instead, playlists simply reference existing songs. This is memory-efficient and mirrors real-world systems.

**UML Diagram**

![](https://pbs.twimg.com/media/HIoppDAbkAAnsst.png)

**Key LLD Concepts Used**

**1\. Encapsulation** All fields are private.

```java
private String title;
```

Access is controlled via getters and methods. This protects internal state.

**2\. Association** A song is associated with an artist.

**3\. Aggregation**

Playlist -\> Songs Playlists use songs but do not own their lifecycle.

**4\. Single Responsibility Principle**

Each class has exactly one job.

Artist - Artist details

Song - Song details

Playlist - Playlist operations

User - User operations

Library - Song storage

**5\. Reusability**

The same song object can be reused across multiple playlists.

```java
workout.addSong(hello);
chill.addSong(hello);
```

**Time Complexity Analysis**

Add song to playlist - O(1) Remove song - O(n) Calculate total duration - O(n) Create playlist - O(1)

Full code -

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

That's all, folks...Cheers!!
