using System;
using System.Collections.Generic;
using System.IO;
using System.Text;

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
        private static double lastTick = 0.0;

        static void Main(string[] args)
        {
            tempLogName = SandboxConfig.DataDirectory + "\\tmp-dump.txt";
            finalLogName = SandboxConfig.DataDirectory + "\\raw-dump.txt";
            Logger.Log(LogLevel.Info, "Writing temp log to {0}", tempLogName);
            if (File.Exists(finalLogName))
            {
                Logger.Log(LogLevel.Info, "Deleting previous raw log file {0}", finalLogName);
                File.Delete(finalLogName);
            }

            FileStream fs = new FileStream(tempLogName, FileMode.Create, FileAccess.Write);
            sw = new StreamWriter(fs);

            AIHeroClient.OnDeath += AIHeroClient_OnDeath;

            AttackableUnit.OnDamage += AttackableUnit_OnDamage;

            Game.OnEnd += Game_OnEnd;
            Game.OnNotify += Game_OnNotify;
            Game.OnTick += Game_OnTick;

            GameObject.OnCreate += GameObject_OnCreate;
            GameObject.OnDelete += GameObject_OnDelete;

            Obj_AI_Base.OnBasicAttack += Obj_AI_Base_OnBasicAttack;
            Obj_AI_Base.OnLevelUp += Obj_AI_Base_OnLevelUp;
            Obj_AI_Base.OnPlayAnimation += Obj_AI_Base_OnPlayAnimation;
            Obj_AI_Base.OnUpdatePosition += Obj_AI_Base_OnUpdatePosition;
            Obj_AI_Base.OnProcessSpellCast += Obj_AI_Base_OnProcessSpellCast;

            Obj_BarracksDampener.OnDamage += Obj_BarracksDampener_OnDamage;
        }

        private static void AIHeroClient_OnDeath(Obj_AI_Base sender, OnHeroDeathEventArgs args)
        {
            logEvent("AIHeroClient_OnDeath", "sender", sender.Name, "sender_id", sender.Index, "network_id", sender.NetworkId);
        }


        private static void AttackableUnit_OnDamage(AttackableUnit sender, AttackableUnitDamageEventArgs args)
        {
            logEvent("AttackableUnit_OnDamage", "sender", sender.Name, "sender_id", sender.Index, "sender_health", sender.Health, "target_health", args.Target.Health, "target_network_id", args.Target.NetworkId);
        }


        private static void Game_OnEnd(GameEndEventArgs args)
        {
            logEvent("Game_OnEnd", "losers", args.LosingTeam, "winners", args.WinningTeam);
        }


        private static void Game_OnNotify(GameNotifyEventArgs args)
        {
            logEvent("Game_OnNotify", "event_id", args.EventId, "network_id", args.NetworkId);
        }


        private static void Game_OnTick(EventArgs args) {
            lastTick = Game.Time;
        }


        private static void GameObject_OnCreate(GameObject sender, EventArgs args)
        {
            if (sender.NetworkId == 0 || sender.Name.EndsWith(".troy"))
            {
                return;
            }
            logEvent("GameObject_OnCreate", "sender", sender.Name, "sender_id", sender.Index, "network_id", sender.NetworkId);
        }


        private static void GameObject_OnDelete(GameObject sender, EventArgs args)
        {
            if (sender.NetworkId == 0 || sender.Name.EndsWith(".troy"))
            {
                return;
            }
            logEvent("GameObject_OnDelete", "sender", sender.Name, "sender_id", sender.Index, "network_id", sender.NetworkId);
        }


        private static void Obj_AI_Base_OnBasicAttack(Obj_AI_Base sender, GameObjectProcessSpellCastEventArgs args)
        {
            logEvent("Obj_AI_Base_OnBasicAttack", "sender", sender.Name, "args", args.ToString());
        }


        private static void Obj_AI_Base_OnLevelUp(Obj_AI_Base sender, Obj_AI_BaseLevelUpEventArgs args)
        {
            logEvent("Obj_AI_Base_OnLevelUp", "sender", sender.Name, "network_id", sender.NetworkId);
        }


        private static void Obj_AI_Base_OnPlayAnimation(Obj_AI_Base sender, GameObjectPlayAnimationEventArgs args)
        {
            logEvent("Obj_AI_Base_OnPlayAnimation", "animation", args.Animation);
        }


        private static void Obj_AI_Base_OnProcessSpellCast(Obj_AI_Base sender, GameObjectProcessSpellCastEventArgs args)
        {
            logEvent("Obj_AI_Base_OnProcessSpellCast", "sender", sender.Name);
        }

        private static void Obj_AI_Base_OnUpdatePosition(Obj_AI_Base sender, Obj_AI_UpdatePositionEventArgs args)
        {
            logEvent("Obj_AI_Base_OnUpdatePosition", "sender", sender.Name, "pos", args.Position.ToString());
        }

        private static void Obj_BarracksDampener_OnDamage(AttackableUnit sender, AttackableUnitDamageEventArgs args)
        {
            logEvent("Obj_BarracksDampener_OnDamage", "sender", sender.Name, "sender_id", sender.Index, "network_id", sender.NetworkId);
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
            Logger.Info("{0}\ttime\t{1}\t{2}", name, Game.Time, sb.ToString());
            sw.WriteLine("{0}\ttime\t{1}\t{2}", name, Game.Time, sb.ToString());
            sw.Flush();
        }
    }
}
