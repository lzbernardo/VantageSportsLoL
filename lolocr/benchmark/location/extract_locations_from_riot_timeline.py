import argparse, json, sys

# This script compares the location information we're generating with the timeline
# information that comes from riot's match api.
# The riot api only emits player locations once a minute, and player locations when they die,
# but we can see how close our location data comes to these data points.
def main():
    parser = argparse.ArgumentParser(description='extract locations from riot timeline')
    parser.add_argument('-i','--input', help='our json input', required=True)
    parser.add_argument('-p','--participant', help='participant id', required=True, type=int)
    parser.add_argument('-m','--match_details', help='riot match details json input', required=True)    
    args = parser.parse_args()

    with open(args.match_details) as in_file, open(args.input) as our_json_in_file:
        data = json.load(in_file)
        known_locations = get_known_locations_from_match_details(data, args.participant)

        data = json.load(our_json_in_file)
        # This tries to find the state corresponding to the time stated in the riot api.
        # It does so by looking for two frames that contain the time stamp.
        # In order to capture events past the last timestamp we have, add a dummy state
        # with a time much larger than anything that should appear
        data = data['states']
        data.append({'match_sec':99999999})

        prev_time = -1
        prev_frame = data[0]
        time_to_look_for_index = 0
        # Since both lists are sorted, we can do one pass and keep an index into the
        # other list.
        for frame in data:
            # If we've found all our data points, we're done
            if time_to_look_for_index >= len(known_locations):
                break
            time_to_look_for = known_locations[time_to_look_for_index]['timestamp'] / float(1000)

            if time_to_look_for > prev_time and time_to_look_for <= frame['match_sec']:
                print "Comparing our location at frame {0}, time {1}: {2},{3} to riot's location {4},{5} Error: {6:.2f}%, {7:.2f}%".format(
                    str(prev_frame['frame_num']),
                    '{0}:{1:02d}'.format(int(time_to_look_for/60), int(time_to_look_for) % 60),
                    str(prev_frame['location']['x']),
                    str(prev_frame['location']['y']),
                    str(known_locations[time_to_look_for_index]['location']['x']),
                    str(known_locations[time_to_look_for_index]['location']['y']),
                    abs(prev_frame['location']['x'] - known_locations[time_to_look_for_index]['location']['x'])/float(150),
                    abs(prev_frame['location']['y'] - known_locations[time_to_look_for_index]['location']['y'])/float(150)
                )
                time_to_look_for_index += 1
                continue
                
            prev_time = frame['match_sec']
            prev_frame = frame

        if time_to_look_for_index != len(known_locations):
            print "Didn't find {0} ({1}/{2})".format(str(time_to_look_for), time_to_look_for_index+1, len(known_locations))

# This should be moved into a utility file
def get_known_locations_from_match_details(match_details, participant_id):
    player_locations = []

    timeline = match_details['timeline']
    for frame in timeline['frames']:
        # The riot api doesn't have a timestamp value for the first frame... Should be 0
        frame_timestamp = frame.get('timestamp', 0)

        if 'participantFrames' in frame:
            frame_location = frame['participantFrames'][str(participant_id)]['position']
        elif 'participantFrameList' in frame:
            for participantFrame in frame['participantFrameList']:
                if participantFrame['participantId'] == participant_id:
                    frame_location = participantFrame['position']
                    break


        # Blank events shouldn't be counted
        if frame_location['x'] == 0 and frame_location['y'] == 0:
            continue
            
        # Look for events that the player did. Some events contain location information
        if 'events' in frame:
            for event in frame['events']:
                # Lots of events have a 0 position, which is probably due to them
                # not being set
                if event['position']['x'] == 0 and event['position']['y'] == 0:
                    continue
                    
                event_participant_id = None
                event_victim_id = None
                if 'participantId' in event:
                    event_participant_id = event['participantId']
                elif 'creatorId' in event:
                    event_participant_id = event['creatorId']

                if 'victimId' in event:
                    event_victim_id = event['victimId']

                if event_participant_id == participant_id or event_victim_id == participant_id:
                    player_locations.append({
                        'timestamp': event['timestamp'],
                        'location': event['position']
                    })

        # Save the player location at every tick
        player_locations.append({
            'timestamp': frame_timestamp,
            'location': frame_location
        })
    
    return player_locations

if __name__ == '__main__':
    main()
