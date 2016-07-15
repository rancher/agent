
def memoize(function):
    memo = {}

    def wrapper(*args):
        if args in memo:
            return memo[args]
        else:
            rv = function(*args)
            memo[args] = rv
            return rv
    return wrapper


def _to_json_object(v):
    if isinstance(v, dict):
        return JsonObject(v)
    elif isinstance(v, list):
        ret = []
        for i in v:
            ret.append(_to_json_object(i))
        return ret
    else:
        return v


class JsonObject:
    def __init__(self, data):
        for k, v in data.items():
            self.__dict__[k] = _to_json_object(v)

    def __getitem__(self, item):
        value = self.__dict__[item]
        if isinstance(value, JsonObject):
            return value.__dict__
        return value

    def __getattr__(self, name):
        return getattr(self.__dict__, name)

    @staticmethod
    def unwrap(json_object):
        if isinstance(json_object, list):
            ret = []
            for i in json_object:
                ret.append(JsonObject.unwrap(i))
            return ret

        if isinstance(json_object, dict):
            ret = {}
            for k, v in json_object.items():
                ret[k] = JsonObject.unwrap(v)
            return ret

        if isinstance(json_object, JsonObject):
            ret = {}
            for k, v in json_object.__dict__.items():
                ret[k] = JsonObject.unwrap(v)
            return ret

        return json_object
