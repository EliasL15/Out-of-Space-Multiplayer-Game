-- phpMyAdmin SQL Dump
-- version 5.2.1
-- https://www.phpmyadmin.net/
--
-- Host: 127.0.0.1:3306
-- Generation Time: Apr 12, 2024 at 02:03 PM
-- Server version: 8.0.35
-- PHP Version: 8.2.4

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `playtest`
--

-- --------------------------------------------------------

--
-- Table structure for table `gameLobbies`
--

CREATE TABLE `gameLobbies` (
  `lobbyPK` int NOT NULL,
  `lobby_timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'time when lobby ends',
  `lobbyID` varchar(8) NOT NULL,
  `alienScore` float DEFAULT NULL,
  `humanScore` float DEFAULT NULL,
  `mvpScore` float DEFAULT NULL,
  `mvpName` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `heatMaps`
--

CREATE TABLE `heatMaps` (
  `lobbyPK` int NOT NULL,
  `lobby_timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'time when lobby ends',
  `lobbyID` varchar(8) NOT NULL,
  `heatmapData` mediumblob NOT NULL COMMENT 'A list of x and y coordinates'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `minigameInfo`
--

CREATE TABLE `minigameInfo` (
  `gameName` varchar(40) NOT NULL,
  `type` enum('1p','1v1','2v2','3v3') CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `minigamePlayers`
--

CREATE TABLE `minigamePlayers` (
  `sessionID` int NOT NULL,
  `name` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `session_timestamp` timestamp NOT NULL,
  `isAstro` tinyint NOT NULL,
  `score` float NOT NULL,
  `won` tinyint NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `minigameSessions`
--

CREATE TABLE `minigameSessions` (
  `sessionID` int NOT NULL,
  `gameName` varchar(50) NOT NULL,
  `lobbyPK` int NOT NULL,
  `timeSpent` float NOT NULL COMMENT 'in seconds',
  `session_timestamp` timestamp NOT NULL COMMENT 'time when minigame ends.'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Indexes for dumped tables
--

--
-- Indexes for table `gameLobbies`
--
ALTER TABLE `gameLobbies`
  ADD PRIMARY KEY (`lobbyPK`) USING BTREE;

--
-- Indexes for table `heatMaps`
--
ALTER TABLE `heatMaps`
  ADD PRIMARY KEY (`lobbyPK`);

--
-- Indexes for table `minigameInfo`
--
ALTER TABLE `minigameInfo`
  ADD PRIMARY KEY (`gameName`);

--
-- Indexes for table `minigamePlayers`
--
ALTER TABLE `minigamePlayers`
  ADD KEY `sessionID` (`sessionID`) USING BTREE;

--
-- Indexes for table `minigameSessions`
--
ALTER TABLE `minigameSessions`
  ADD PRIMARY KEY (`sessionID`),
  ADD KEY `gameName` (`gameName`),
  ADD KEY `lobbyPK` (`lobbyPK`);

--
-- AUTO_INCREMENT for dumped tables
--

--
-- AUTO_INCREMENT for table `gameLobbies`
--
ALTER TABLE `gameLobbies`
  MODIFY `lobbyPK` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `heatMaps`
--
ALTER TABLE `heatMaps`
  MODIFY `lobbyPK` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `minigamePlayers`
--
ALTER TABLE `minigamePlayers`
  MODIFY `sessionID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `minigameSessions`
--
ALTER TABLE `minigameSessions`
  MODIFY `sessionID` int NOT NULL AUTO_INCREMENT;

--
-- Constraints for dumped tables
--

--
-- Constraints for table `heatMaps`
--
ALTER TABLE `heatMaps`
  ADD CONSTRAINT `heatMaps_ibfk_1` FOREIGN KEY (`lobbyPK`) REFERENCES `gameLobbies` (`lobbyPK`);

--
-- Constraints for table `minigamePlayers`
--
ALTER TABLE `minigamePlayers`
  ADD CONSTRAINT `minigamePlayers_ibfk_1` FOREIGN KEY (`sessionID`) REFERENCES `minigameSessions` (`sessionID`);

--
-- Constraints for table `minigameSessions`
--
ALTER TABLE `minigameSessions`
  ADD CONSTRAINT `minigameSessions_ibfk_1` FOREIGN KEY (`gameName`) REFERENCES `minigameInfo` (`gameName`),
  ADD CONSTRAINT `minigameSessions_ibfk_2` FOREIGN KEY (`lobbyPK`) REFERENCES `gameLobbies` (`lobbyPK`);
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
