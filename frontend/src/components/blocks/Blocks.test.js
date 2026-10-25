import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/svelte';
import {
  GameHeroBackdrop,
  GameLogo,
  GameSynopsis,
  GenrePillGroup,
  LiveCommunityBadge,
  PlayActionCluster,
  MultiseatStatusPill,
  CloudSaveIndicator,
  AchievementRibbon,
  PatchNotesFeed,
  CoverCard,
  CompactGameRow,
  SearchFilterBar,
  TaskStatusBar,
  CarouselShelf
} from './index.js';

describe('Universal UI Blocks Catalog', () => {
  describe('GameHeroBackdrop', () => {
    it('renders hero image when heroUrl provided', () => {
      render(GameHeroBackdrop, {
        heroUrl: 'https://example.com/hero.jpg',
        glowColor: 'rgba(255, 0, 0, 0.2)'
      });
      expect(screen.getByTestId('game-hero-backdrop')).toBeInTheDocument();
      const img = screen.getByAltText('Hero Backdrop');
      expect(img).toHaveAttribute('src', 'https://example.com/hero.jpg');
    });

    it('falls back cleanly to dark canvas when no heroUrl', () => {
      render(GameHeroBackdrop, {
        heroUrl: ''
      });
      expect(screen.getByTestId('game-hero-backdrop')).toBeInTheDocument();
      expect(screen.queryByAltText('Hero Backdrop')).toBeNull();
    });
  });

  describe('GameLogo', () => {
    it('renders logo image when logoUrl provided', () => {
      render(GameLogo, {
        logoUrl: 'https://example.com/logo.png',
        title: 'Hades II'
      });
      const img = screen.getByAltText('Hades II');
      expect(img).toHaveAttribute('src', 'https://example.com/logo.png');
    });

    it('falls back to stylized text title when no logoUrl', () => {
      render(GameLogo, {
        logoUrl: '',
        title: 'Cyberpunk 2077'
      });
      expect(screen.getByText('Cyberpunk 2077')).toBeInTheDocument();
    });
  });

  describe('GameSynopsis', () => {
    it('renders cleaned description, developer, publisher, and release date', () => {
      render(GameSynopsis, {
        description: '<p>A &quot;legendary&quot; adventure.</p>',
        developers: ['Supergiant Games'],
        publishers: ['Supergiant Games'],
        releaseDate: 'May 6, 2024',
        buildId: '1420985'
      });
      expect(screen.getByText('A "legendary" adventure.')).toBeInTheDocument();
      expect(screen.getByText('Supergiant Games')).toBeInTheDocument();
      expect(screen.getByText('May 6, 2024')).toBeInTheDocument();
      expect(screen.getByText('1420985')).toBeInTheDocument();
    });
  });

  describe('GenrePillGroup', () => {
    it('renders genre pills and handles select', async () => {
      const onSelect = vi.fn();
      render(GenrePillGroup, {
        genres: ['Action', 'Roguelike', 'Indie'],
        selectedGenre: 'Action',
        onselect: onSelect
      });
      const pill = screen.getByText('Roguelike');
      expect(pill).toBeInTheDocument();

      await fireEvent.click(pill);
      expect(onSelect).toHaveBeenCalledTimes(1);
      expect(onSelect).toHaveBeenCalledWith({ genre: 'Roguelike' });
    });
  });

  describe('LiveCommunityBadge', () => {
    it('renders live player count with formatting', () => {
      render(LiveCommunityBadge, {
        playerCount: 24512
      });
      expect(screen.getByTestId('live-community-badge')).toBeInTheDocument();
      expect(screen.getByText('24,512')).toBeInTheDocument();
      expect(screen.getByText('playing right now')).toBeInTheDocument();
    });

    it('renders nothing when count is 0', () => {
      render(LiveCommunityBadge, {
        playerCount: 0
      });
      expect(screen.queryByTestId('live-community-badge')).toBeNull();
    });
  });

  describe('PlayActionCluster', () => {
    it('renders Play Now when ready and installed, triggers onplay', async () => {
      const onPlay = vi.fn();
      render(PlayActionCluster, {
        isInstalled: true,
        isReady: true,
        isRunning: false,
        onplay: onPlay
      });
      const playBtn = screen.getByText('Play Now');
      expect(playBtn).toBeInTheDocument();

      await fireEvent.click(playBtn);
      expect(onPlay).toHaveBeenCalledTimes(1);
    });

    it('renders Stop Process when running, triggers onstop', async () => {
      const onStop = vi.fn();
      render(PlayActionCluster, {
        isInstalled: true,
        isRunning: true,
        pid: 5821,
        onstop: onStop
      });
      const stopBtn = screen.getByText('Stop Process (#5821)');
      expect(stopBtn).toBeInTheDocument();

      await fireEvent.click(stopBtn);
      expect(onStop).toHaveBeenCalledTimes(1);
    });

    it('triggers onconfigure when gear icon clicked', async () => {
      const onConfigure = vi.fn();
      render(PlayActionCluster, {
        isInstalled: true,
        isReady: true,
        onconfigure: onConfigure
      });
      const gearBtn = screen.getByTitle('Tuning & Compatibility Options');

      await fireEvent.click(gearBtn);
      expect(onConfigure).toHaveBeenCalledTimes(1);
    });
  });

  describe('MultiseatStatusPill', () => {
    it('renders 2P Active when session is running', () => {
      render(MultiseatStatusPill, {
        isCoop: true,
        session: { is_running: true }
      });
      expect(screen.getByText('2P Active')).toBeInTheDocument();
    });

    it('renders Co-op Armed when armed', () => {
      render(MultiseatStatusPill, {
        isCoop: true,
        p1Gamepad: 'DualSense',
        p2Gamepad: 'Xbox'
      });
      expect(screen.getByText('Co-op Armed')).toBeInTheDocument();
      expect(screen.getByText('P1: DualSense | P2: Xbox')).toBeInTheDocument();
    });
  });

  describe('CloudSaveIndicator', () => {
    it('renders backup count and triggers restore', async () => {
      const onRestore = vi.fn();
      render(CloudSaveIndicator, {
        backupCount: 4,
        lastBackupTime: Math.floor(Date.now() / 1000) - 120,
        onrestore: onRestore
      });
      expect(screen.getByText('4 Backups')).toBeInTheDocument();

      await fireEvent.click(screen.getByText('4 Backups'));
      expect(onRestore).toHaveBeenCalledTimes(1);
    });
  });

  describe('AchievementRibbon', () => {
    it('renders achievement percentage and badge icons', async () => {
      const onSelect = vi.fn();
      const mockAch = [
        { id: '1', name: 'First Blood', unlocked: true, icon_url: 'https://example.com/ach1.png' },
        { id: '2', name: 'Master', unlocked: false, icon_gray_url: 'https://example.com/ach2.png' }
      ];
      render(AchievementRibbon, {
        achievements: mockAch,
        onselect: onSelect
      });
      expect(screen.getByText('1 / 2 (50%)')).toBeInTheDocument();
      expect(screen.getByText('First Blood')).toBeInTheDocument();

      await fireEvent.click(screen.getByText('First Blood'));
      expect(onSelect).toHaveBeenCalledTimes(1);
      expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ achievement: expect.objectContaining({ id: '1' }) }));
    });
  });

  describe('PatchNotesFeed', () => {
    it('renders news items and strips formatting tags', async () => {
      const onRead = vi.fn();
      const mockNews = [
        { title: 'Patch 1.04 Released', date: 1715000000, contents: '[b]Bug fixes[/b] and performance improvements.', feedlabel: 'Community' }
      ];
      render(PatchNotesFeed, {
        newsItems: mockNews,
        onread: onRead
      });
      expect(screen.getByText('Patch 1.04 Released')).toBeInTheDocument();
      expect(screen.getByText('Bug fixes and performance improvements.')).toBeInTheDocument();

      await fireEvent.click(screen.getByText('Patch 1.04 Released'));
      expect(onRead).toHaveBeenCalledTimes(1);
    });
  });

  describe('CoverCard', () => {
    it('renders poster card with title and triggers click without size or playtime', async () => {
      const onClick = vi.fn();
      render(CoverCard, {
        name: 'Hollow Knight',
        appId: '367520',
        selected: true,
        onclick: onClick
      });
      expect(screen.getByText('Hollow Knight')).toBeInTheDocument();
      expect(screen.queryByText(/hrs|mins|GiB|MiB/)).not.toBeInTheDocument();

      await fireEvent.click(screen.getByText('Hollow Knight'));
      expect(onClick).toHaveBeenCalledTimes(1);
    });
  });

  describe('CompactGameRow', () => {
    it('renders tabular row and handles play dispatch', async () => {
      const onPlay = vi.fn();
      const mockGame = {
        name: 'Elden Ring',
        platform: 'win64',
        diskSizeBytes: 60000000000,
        playtimeSeconds: 36000,
        runner: 'UMU-Proton'
      };
      render(CompactGameRow, {
        game: mockGame,
        onplay: onPlay
      });
      expect(screen.getByText('Elden Ring')).toBeInTheDocument();
      expect(screen.getByText('UMU-Proton')).toBeInTheDocument();

      await fireEvent.click(screen.getByText('Play'));
      expect(onPlay).toHaveBeenCalledTimes(1);
    });
  });

  describe('SearchFilterBar', () => {
    it('handles input when typing and clears on clear click', async () => {
      const onInput = vi.fn();
      render(SearchFilterBar, {
        query: 'Hades',
        totalCount: 50,
        filteredCount: 2,
        oninput: onInput
      });
      const input = screen.getByPlaceholderText('Search titles, genres, or developers...');
      expect(input.value).toBe('Hades');

      await fireEvent.input(input, { target: { value: 'Hades II' } });
      expect(onInput).toHaveBeenCalledWith({ query: 'Hades II' });

      const clearBtn = screen.getByTitle('Clear Search (Esc)');
      await fireEvent.click(clearBtn);
      expect(onInput).toHaveBeenCalledWith({ query: '' });
    });
  });

  describe('TaskStatusBar', () => {
    it('renders active task and speed, dispatches toggle', async () => {
      const onToggle = vi.fn();
      const mockTasks = [
        { operation: 'unpack', gameName: 'Balatro', progress: 75 }
      ];
      render(TaskStatusBar, {
        activeTasks: mockTasks,
        totalSpeedBps: 15000000,
        isExpanded: false,
        ontoggle: onToggle
      });
      expect(screen.getByText(/unpack:/i)).toBeInTheDocument();
      expect(screen.getByText('Balatro')).toBeInTheDocument();
      expect(screen.getByText('(75%)')).toBeInTheDocument();

      await fireEvent.click(screen.getByText('View Queue'));
      expect(onToggle).toHaveBeenCalledTimes(1);
    });
  });

  describe('CarouselShelf', () => {
    it('renders list of cover cards and handles select', async () => {
      const onSelect = vi.fn();
      const mockGames = [
        { name: 'Game A', appId: '1' },
        { name: 'Game B', appId: '2' }
      ];
      render(CarouselShelf, {
        games: mockGames,
        selectedIndex: 0,
        onselect: onSelect
      });
      expect(screen.getByText('Game A')).toBeInTheDocument();
      expect(screen.getByText('Game B')).toBeInTheDocument();

      await fireEvent.click(screen.getByText('Game B'));
      expect(onSelect).toHaveBeenCalledTimes(1);
      expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ index: 1 }));
    });
  });
});
