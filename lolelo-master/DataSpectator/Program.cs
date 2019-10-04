using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Threading;

namespace DataSpectator
{
    using EloBuddy;
    using EloBuddy.Sandbox;
    using EloBuddy.SDK;
    using EloBuddy.SDK.Enumerations;
    using EloBuddy.SDK.Utils;

    class Program
    {
        private static StreamWriter sw;
        private static string tempLogName;
        private static string finalLogName;

        static void Main(string[] args)
        {
            tempLogName = SandboxConfig.DataDirectory + "\\tmp-events.txt";
            finalLogName = SandboxConfig.DataDirectory + "\\raw-events.txt";
            Logger.Info("Writing temp log to {0}", tempLogName);
            if (File.Exists(finalLogName))
            {
                Logger.Info("Deleting previous raw log file {0}", finalLogName);
                File.Delete(finalLogName);
            }

            FileStream fs = new FileStream(tempLogName, FileMode.Create, FileAccess.Write);
            sw = new StreamWriter(fs);

            AttackableUnit.OnDamage += AttackableUnit_OnDamage;

            Game.OnNotify += Game_OnNotify;
            Game.OnTick += OnTick;
            Game.OnEnd += OnEnd;
            
            GameObject.OnCreate += GameObject_OnCreate;
            GameObject.OnDelete += GameObject_OnDelete;

            Obj_AI_Base.OnBasicAttack += Obj_AI_Base_OnBasicAttack;
            Obj_AI_Base.OnLevelUp += Obj_AI_Base_OnLevelUp;
            Obj_AI_Base.OnProcessSpellCast += Obj_AI_Base_OnProcessSpellCast;
            // As of 6.22, we no longer see Obj_AI_Base.OnPlayAnimation events
            // fire. If this changes see what this event handler used to do in
            // commit a3c0e1

            ThreadStart childref = new ThreadStart(ChildThreadFunction);
            Thread childThread = new Thread(childref);
            childThread.Start();
        }

        // gameOver is set to true when we detect that a game has ended/stalled permanently.
        private static bool gameOver = false;
        // nextPingTime is the next second at which we want to emit a PING event.
        private static double nextPingTime = 0;
        // nextEndCheckTime is the next second at which we want to check if the game is over.
        private static double nextEndCheckTime = 0;
        // lastAttackTime is used as a proxy for whether a game is still "active". If the game ended via
        // surrender, neither nexus will be destroyed.
        private static double lastAttackTime = 0.0;
        // lastChampPositions is used as a proxy for whether a game is still "active".
        private static string lastChampPositions = "";
        private static string pastChampPositions = "";
        private static double pastChampPositionsTime = 0.0;
        private static double samePositionsTime = 0.0;

        // nameByNetworkId stores the UTF-8 name of each hero by their networkId. This is so that we don't have
        // to convert from ascii->utf8 at every ping.
        private static Dictionary<uint, string> nameByNetworkId = new Dictionary<uint, string>();
        // deadByNetworkId keeps track of who is dead at any given time. This is useful because some champs
        // play multiple death animations per kill, so keeping this state helps us not emit multiple death
        // events per champ death.
        private static Dictionary<uint, bool> deadByNetworkId = new Dictionary<uint, bool>();

        private static string heroNameOrDefault(uint index, string orig)
        {
            if (nameByNetworkId.ContainsKey(index))
            {
                return nameByNetworkId[index];
            }
            // Even when the hero index is not in the map, this name may still
            // be the name of a hero. For instance, when wukong emits a decoy,
            // the attack target name is set to the summoner name, but the index
            // is an ephemeral one... so we just always wrap this string in
            // quotation marks to be safe.
            return  "\"" + orig + "\"";
        }

        private static void RememberNetworkIds()
        {
            if (EntityManager.Heroes.AllHeroes.Count < 1)
            {
                return;
            }
            for (int i = 0; i < EntityManager.Heroes.AllHeroes.Count; i++)
            {
                AIHeroClient hero = EntityManager.Heroes.AllHeroes[i];
                uint nId = (uint)hero.NetworkId;

                // The default encoding is ascii. Convert the hero name to utf8
                byte[] bytes = Encoding.Default.GetBytes(hero.Name);
                string name = "\"" + Encoding.UTF8.GetString(bytes) + "\"";
                nameByNetworkId[nId] = name;

                logEvent("ID_HERO", "name", name, "network_id", nId);
            }

            foreach (var turret in ObjectManager.Get<Obj_AI_Turret>())
            {
                logEvent("ID_TURRET", "name", turret.Name, "network_id", (uint)turret.NetworkId);
            }

            foreach (var barracks in ObjectManager.Get<Obj_BarracksDampener>())
            {
                logEvent("ID_BARRACKS", "name", barracks.Name, "network_id", (uint)barracks.NetworkId);
            }
        }

        /**
         * Game_OnNotify is used to detect events that we can't register handlers for.
         * Currently the only such event is an HQ death, which - in the event of a
         * surrender - is the only way we can detect that the game has ended.
         */
        private static void Game_OnNotify(GameNotifyEventArgs args)
        {
            switch (args.EventId)
            {
                case GameEventId.OnChampionDie:
                    logEvent("CHAMP_DIE", "time", Game.Time, "network_id", args.NetworkId);
                    break;
                case GameEventId.OnChampionKill:
                    logEvent("CHAMP_KILL", "time", Game.Time, "network_id", args.NetworkId);
                    break;
                case GameEventId.OnChampionLevelUp:
                    logEvent("CHAMP_LEVEL_UP", "time", Game.Time, "network_id", args.NetworkId);
                    break;
                case GameEventId.OnDampenerRespawnSoon:
                    logEvent("DAMPENER_RESPAWN_SOON", "time", Game.Time, "network_id", args.NetworkId);
                    break;
                case GameEventId.OnDampenerRespawn:
                    logEvent("DAMPENER_RESPAWN", "time", Game.Time, "network_id", args.NetworkId);
                    break;
                case GameEventId.OnDie:
                    logEvent("DIE", "time", Game.Time, "network_id", args.NetworkId);
                    break;
                case GameEventId.OnEndGame:
                    logEvent("END_GAME", "time", Game.Time);
                    // NOTE: This does not appear to be the actual EndGame, so
                    // we don't set gameOver to true. Perhaps this is when the
                    // full replay is downloaded, so the client is aware of
                    // when the end of the game is?
                    break;
                case GameEventId.OnKill:
                    logEvent("KILL", "time", Game.Time, "network_id", args.NetworkId);
                    break;
                case GameEventId.OnSurrenderVote:
                    logEvent("SURRENDER_VOTE", "time", Game.Time, "network_id", args.NetworkId);
                    break;
                case GameEventId.OnSurrenderAgreed:
                    logEvent("SURRENDER_AGREED", "time", Game.Time, "network_id", args.NetworkId);
                    break;
            }
        }

        /**
         * Tick events seem to happen 25 times per second regardless of the playback speed,
         * whereas update events seem to occur with much more variable frequency. So we use
         * OnTick to log our "ping" events, making sure that we only log a certain number
         * of updates per second.
         */
        private static void OnTick(EventArgs args)
        {
            if (Game.Time >= nextPingTime)
            {
                if (nameByNetworkId.Count == 0)
                {
                    RememberNetworkIds();
                }

                // Emit a ping every half second.
                while (Game.Time >= nextPingTime)
                {
                    nextPingTime += 0.5;
                }

                // Check champ positions, to see if the game has stalled
                lastChampPositions = SerializeChampPositions(EntityManager.Heroes.AllHeroes);
                if ( lastChampPositions == pastChampPositions )
                {
                    samePositionsTime += Game.Time - pastChampPositionsTime;
                }
                else
                {
                    samePositionsTime = 0.0;
                }
                pastChampPositions = lastChampPositions;
                pastChampPositionsTime = Game.Time;

                for (int i = 0; i < EntityManager.Heroes.AllHeroes.Count; i++)
                {
                    AIHeroClient hero = EntityManager.Heroes.AllHeroes[i];

                    SpellDataInst Q = hero.Spellbook.GetSpell(SpellSlot.Q);
                    SpellDataInst W = hero.Spellbook.GetSpell(SpellSlot.W);
                    SpellDataInst E = hero.Spellbook.GetSpell(SpellSlot.E);
                    SpellDataInst R = hero.Spellbook.GetSpell(SpellSlot.R);
                    SpellDataInst S1 = hero.Spellbook.GetSpell(SpellSlot.Summoner1);
                    SpellDataInst S2 = hero.Spellbook.GetSpell(SpellSlot.Summoner2);

                    uint nId = (uint)hero.NetworkId;
                    logEvent(
                        "PING",
                        "time", Game.Time,
                        "index", hero.Index,
                        "network_id", nId,
                        "name", nameByNetworkId[nId],
                        "level", hero.Level,
                        "position", hero.Position.ToString(),
                        "in_grass", hero.Position.IsGrass(),
                        "gold", hero.Gold,
                        // "minions_killed", hero.MinionsKilled,
                        // "neutral_minions_killed", hero.NeutralMinionsKilled,
                        "dead", hero.IsDead,
                        "under_turret", hero.IsUnderTurret(),
                        "under_enemy_turret", hero.IsUnderEnemyturret(),
                        "health", hero.IsDead ? 0 : hero.Health,
                        "health_max", hero.MaxHealth,
                        "mana", hero.Mana,
                        "mana_max", hero.MaxMana,
                        "q_level", Q.Level,
                        "q_cooldown", Q.IsOnCooldown,
                        "q_exp", Q.CooldownExpires,
                        "w_level", W.Level,
                        "w_cooldown", W.IsOnCooldown,
                        "w_exp", W.CooldownExpires,
                        "e_level", E.Level,
                        "e_cooldown", E.IsOnCooldown,
                        "e_exp", E.CooldownExpires,
                        "r_level", R.Level,
                        "r_cooldown", R.IsOnCooldown,
                        "r_exp", R.CooldownExpires,
                        "s1_level", S1.Level,
                        "s1_cooldown", S1.IsOnCooldown,
                        "s1_exp", S1.CooldownExpires,
                        "s2_level", S2.Level,
                        "s2_cooldown", S2.IsOnCooldown,
                        "s2_exp", S2.CooldownExpires);

                    deadByNetworkId[nId] = hero.IsDead;
                }
            }

            if (Game.Time > nextEndCheckTime)
            {
                nextEndCheckTime = Game.Time + 10;
                // we'll exit at the end of this block if:
                // a) we noticed that an HQ is dead (even if we can't tell which one)
                // b) notice (below) that a nexus has been destroyed
                // c) haven't seen any attacks for more than 3 minutes
                // d) no champions have moved in 3 minutes
                if (lastAttackTime > 0 && (Game.Time - lastAttackTime > 180))
                {
                    logEvent("GAME_STALL", "time", Game.Time, "last_attack_time", lastAttackTime);
                    gameOver = true;
                }

                // sometimes, the game will stall with a neutral monster stuck in an auto-attack loop.
                // This causes the above stall check to never trigger, so we have another stall check here.
                // If all 10 champions haven't moved in 30 seconds, report a game stall.
                if (samePositionsTime > 30)
                {
                    logEvent("GAME_STALL", "time", Game.Time, "same_positions_time", samePositionsTime);
                    gameOver = true;
                }

                foreach (var hq in ObjectManager.Get<Obj_HQ>())
                {
                    if (hq.Health == 0 || hq.IsDead)
                    {
                        logEvent("NEXUS_DESTROYED", "nexus", hq.Name, "time", Game.Time);
                        gameOver = true;
                    }
                }

                if (gameOver)
                {
                    OnEnd(null);
                }
            }
        }

        public static void OnEnd(GameEndEventArgs args)
        {
            logEvent("GAME_END", "time", Game.Time);
            sw.Close();

            Logger.Info("Moving temp log to {0}", finalLogName);
            File.Move(tempLogName, finalLogName);

            Game.QuitGame();
        }

        private static void Obj_AI_Base_OnLevelUp(Obj_AI_Base sender, Obj_AI_BaseLevelUpEventArgs args)
        {
            // NOTE: args.Level == 0 most likely means a respawn rather than
            // anything resembling a champion level up, but we handle that case
            // in post-processing.
            uint nId = (uint)sender.NetworkId;
            logEvent("LEVEL_UP",
                "time", Game.Time,
                "sender", heroNameOrDefault(nId, sender.Name),
                "network_id", nId,
                "level", args.Level.ToString());
        }

        private static void Obj_AI_Base_OnBasicAttack(Obj_AI_Base sender, GameObjectProcessSpellCastEventArgs args)
        {
            if (sender == null || sender.IsMinion || args == null || args.Target == null)
            {
                return;
            }
            // sometimes after surrender minions keep auto-attacking forever, so make sure not to increment
            // last attack time until we know that the sender was not a minion.
            lastAttackTime = Game.Time;
            uint senderNId = (uint)sender.NetworkId;
            uint targetNId = (uint)args.Target.NetworkId;
            logEvent("BASIC_ATTACK",
                "time", Game.Time,
                "sender", heroNameOrDefault(senderNId, sender.Name),
                "network_id", senderNId,
                "target", heroNameOrDefault(targetNId, args.Target.Name),
                "target_network_id", targetNId,
                "target_position", args.Target.Position.ToString());
        }

        private static void AttackableUnit_OnDamage(AttackableUnit sender, AttackableUnitDamageEventArgs args)
        {
            // Include only damage to any champ.
            if (args.Target.Type != GameObjectType.AIHeroClient)
            {
                return;
            }
            uint senderNId = (uint)sender.NetworkId;
            uint targetNId = (uint)args.Target.NetworkId;
            logEvent("DAMAGE",
                "time", Game.Time,
                "sender", heroNameOrDefault(senderNId, sender.Name),
                "network_id", sender.NetworkId,
                "damage", args.Damage,
                "target", heroNameOrDefault(targetNId, args.Target.Name),
                "target_network_id", targetNId,
                "hit_type", args.HitType.ToString(),
                "type", args.Type);
        }

        /**
         * SpellCasts represent pretty much any time a player presses a button.
         * Abilities, Summoner Spells, Items, Wards, Self cast abilities, targeted abilities, and skillshots all get picked up here
         * This is different from "OnSpellCast", which appears to only be projectile-based abilities
         * Note that this will also have a lot of extra thingsthat aren't player button presses. For example, Jhin's basic attacks
         * count as spells because there's special mechanics behind them. We'll have to distill these for each champion.
         * */
        private static void Obj_AI_Base_OnProcessSpellCast(Obj_AI_Base sender, GameObjectProcessSpellCastEventArgs args)
        {
            lastAttackTime = Game.Time;
            if (sender.Type != GameObjectType.AIHeroClient)
            {
                return;
            }
            uint nId = (uint)sender.NetworkId;
            logEvent("SPELL_CAST",
                "time", Game.Time,
                "sender", heroNameOrDefault(nId, sender.Name),
                "network_id", nId,
                "name", args.SData.Name,
                "slot", "\"" + args.Slot.ToString() + "\"",  // need quotes because sometimes the string value is e.g. "46"
                "level", args.Level,
                "start_position", args.Start == null ? "null" : args.Start.ToString(),
                "end_position", args.End == null ? "null" : args.End.ToString(),
                "target", args.Target == null ? "null" : heroNameOrDefault((uint)args.Target.NetworkId, args.Target.Name),
                "target_network_id", args.Target == null ? 0 : (uint)args.Target.NetworkId);
        }

        private static void GameObject_OnCreate(GameObject sender, EventArgs args)
        {
            uint nId = (uint)sender.NetworkId;
            // Items without a network id aren't really useful to us (e.g.
            // missles), nor are the "troy" items, which as best I can tell are
            // particles rendered on screen during attacks/spells/etc.
            if (sender.NetworkId == 0 || sender.Name.EndsWith(".troy"))
            {
                return;
            }
            logEvent("ON_CREATE",
                "time", Game.Time,
                "sender", heroNameOrDefault(nId, sender.Name),
                "team_id", (int)sender.Team,
                "network_id", nId,
                "position", sender.Position.ToString());
        }

        private static void GameObject_OnDelete(GameObject sender, EventArgs args)
        {
            uint nId = (uint)sender.NetworkId;
            // See comment in OnCreate for reason we filter these out.
            if (nId == 0 || sender.Name.EndsWith(".troy"))
            {
                return;
            }
            logEvent("ON_DELETE",
                "time", Game.Time,
                "sender", heroNameOrDefault(nId, sender.Name),
                "team_id", (int)sender.Team,
                "network_id", nId,
                "position", sender.Position.ToString());
        }

        private static string SerializeChampPositions(List<AIHeroClient> heroes)
        {
            string s = "";
            for (int i = 0; i < heroes.Count; i++)
            {
                s += heroes[i].Position.ToString();
            }
            return s;
        }

        /**
         * logEvent writes a single event to the configured StreamWriter. Each event should
         * contain an even number of params (name->value) not including the name string.
         * e.g. logEvent("FOO_EVENT", "name", "Billy", "age", 20, "alive", True);
         */
        private static void logEvent(string name, params object[] props)
        {
            var sb = new StringBuilder();
            for (var i = 0; i < props.Length; i++)
            {
                if (props[i] == null)
                {
                    sb.Append("null");
                }
                else if (props[i].GetType() == typeof(bool))
                {
                    // Lowercase boolean values.
                    sb.AppendFormat("{0}", props[i].ToString().ToLower());
                }
                else if (props[i].GetType() == typeof(float))
                {
                    // prevent small floats from being formatted in scientific notation.
                    float val = (float)props[i];
                    sb.Append((val > 0.0 && val < 0.0001) ? 0.0001 : val);
                }
                else
                {
                    // Everything else (strings/ints/floats) just append as is.
                    sb.Append(props[i]);
                }

                // Tab-delimit fields. Spaces are difficult because some values contain them.
                sb.Append("\t");
            }
            sw.WriteLine("{0}\t{1}", name, sb.ToString());
            sw.Flush();
        }

        public static void ChildThreadFunction()
        {
	    int timeBeforeFirstAttack = 0;
	    while (true)
            {
                if (lastAttackTime > 0)
                {
                    string champPositions = SerializeChampPositions(EntityManager.Heroes.AllHeroes);
                    Thread.Sleep(10000);
                    string newChampPositions = SerializeChampPositions(EntityManager.Heroes.AllHeroes);


                    if ( champPositions == newChampPositions )
                    {
                        // Console.WriteLine("Same champ positions for 10 seconds");
                        OnEnd(null);
                    }
                }
                Thread.Sleep(1000);
		
		if (lastAttackTime == 0)
		{
		    timeBeforeFirstAttack += 1000;

		    if (timeBeforeFirstAttack > 180000)
		    {
		        OnEnd(null);
		    }
		}
            }
        }
    }
}
